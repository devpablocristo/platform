package lifecycle

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/devpablocristo/platform/errors/go/domainerr"
	"github.com/google/uuid"
)

// fakeRepo implements RepositoryPort with an in-memory map.
type fakeRepo struct {
	rows         map[uuid.UUID]*time.Time // key = resourceID, value = archivedAt
	lastTenantID string
	failNotFound bool
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{rows: make(map[uuid.UUID]*time.Time)}
}

func (r *fakeRepo) seed(id uuid.UUID) { r.rows[id] = nil }

func (r *fakeRepo) SoftDelete(_ context.Context, tenantID string, id uuid.UUID, at time.Time) error {
	r.lastTenantID = tenantID
	v, ok := r.rows[id]
	if !ok || v != nil {
		return domainerr.NotFoundf("fake", id.String())
	}
	t := at
	r.rows[id] = &t
	return nil
}

func (r *fakeRepo) Restore(_ context.Context, tenantID string, id uuid.UUID) error {
	r.lastTenantID = tenantID
	v, ok := r.rows[id]
	if !ok || v == nil {
		return domainerr.NotFoundf("fake", id.String())
	}
	r.rows[id] = nil
	return nil
}

func (r *fakeRepo) HardDelete(_ context.Context, tenantID string, id uuid.UUID) error {
	r.lastTenantID = tenantID
	if _, ok := r.rows[id]; !ok {
		return domainerr.NotFoundf("fake", id.String())
	}
	delete(r.rows, id)
	return nil
}

func (r *fakeRepo) IsArchived(_ context.Context, tenantID string, id uuid.UUID) (bool, error) {
	r.lastTenantID = tenantID
	v, ok := r.rows[id]
	if !ok {
		return false, domainerr.NotFoundf("fake", id.String())
	}
	return v != nil, nil
}

// recordingAudit captures appended entries for assertions.
type recordingAudit struct{ entries []ArchiveAudit }

func (a *recordingAudit) Append(_ context.Context, e ArchiveAudit) error {
	a.entries = append(a.entries, e)
	return nil
}

func newTestService(t *testing.T, policies ...*ArchivePolicy) (*Service, *fakeRepo, *recordingAudit) {
	t.Helper()
	repo := newFakeRepo()
	audit := &recordingAudit{}
	registry := NewStaticPolicyRegistry(policies...)
	repos := map[string]RepositoryPort{}
	for _, p := range policies {
		repos[p.ResourceType] = repo
	}
	svc, err := NewServiceWithRepos(repos, audit, registry,
		WithClock(func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) }),
	)
	if err != nil {
		t.Fatalf("NewServiceWithRepos: %v", err)
	}
	return svc, repo, audit
}

func TestSoftDelete_HappyPath(t *testing.T) {
	policy := &ArchivePolicy{
		ResourceType:    "widget",
		AllowArchive:    true,
		AllowHardDelete: true,
		RequireReason:   false,
	}
	svc, repo, audit := newTestService(t, policy)

	id := uuid.New()
	repo.seed(id)

	err := svc.SoftDelete(context.Background(), &ArchiveRequest{
		ResourceType: "widget",
		ResourceID:   id,
		TenantID:     "argos-local-org",
		Actor:        "tester@example.com",
		Reason:       "ad-hoc",
	})
	if err != nil {
		t.Fatalf("SoftDelete: %v", err)
	}
	if repo.rows[id] == nil {
		t.Fatalf("expected row to be archived")
	}
	if repo.lastTenantID != "argos-local-org" {
		t.Fatalf("expected opaque tenant string, got %q", repo.lastTenantID)
	}
	if len(audit.entries) != 1 || audit.entries[0].Action != ActionArchive {
		t.Fatalf("expected 1 archive audit entry, got %+v", audit.entries)
	}
}

func TestSoftDelete_RequiresReason(t *testing.T) {
	policy := &ArchivePolicy{ResourceType: "widget", AllowArchive: true, RequireReason: true}
	svc, repo, _ := newTestService(t, policy)
	id := uuid.New()
	repo.seed(id)

	err := svc.SoftDelete(context.Background(), &ArchiveRequest{
		ResourceType: "widget",
		ResourceID:   id,
		TenantID:     "argos-local-org",
	})
	if !errors.Is(err, ErrReasonRequired) {
		t.Fatalf("expected ErrReasonRequired, got %v", err)
	}
}

func TestSoftDelete_BlockedByValidator(t *testing.T) {
	sentinel := errors.New("has active relations")
	policy := &ArchivePolicy{
		ResourceType: "widget",
		AllowArchive: true,
		ValidateRelations: func(_ context.Context, _ string, _ uuid.UUID) error {
			return sentinel
		},
	}
	svc, repo, audit := newTestService(t, policy)
	id := uuid.New()
	repo.seed(id)

	err := svc.SoftDelete(context.Background(), &ArchiveRequest{
		ResourceType: "widget",
		ResourceID:   id,
		TenantID:     "argos-local-org",
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected sentinel error, got %v", err)
	}
	if repo.rows[id] != nil {
		t.Fatalf("row should not have been archived")
	}
	if len(audit.entries) != 0 {
		t.Fatalf("no audit should be appended on rejection")
	}
}

func TestRestore_HappyPath(t *testing.T) {
	policy := &ArchivePolicy{ResourceType: "widget", AllowArchive: true}
	svc, repo, audit := newTestService(t, policy)
	id := uuid.New()
	repo.seed(id)
	_ = svc.SoftDelete(context.Background(), &ArchiveRequest{
		ResourceType: "widget",
		ResourceID:   id,
		TenantID:     "argos-local-org",
	})

	if err := svc.Restore(context.Background(), &RestoreRequest{
		ResourceType: "widget",
		ResourceID:   id,
		TenantID:     "argos-local-org",
		Actor:        "tester@example.com",
	}); err != nil {
		t.Fatalf("Restore: %v", err)
	}
	if repo.rows[id] != nil {
		t.Fatalf("row should not be archived after Restore")
	}
	if len(audit.entries) != 2 || audit.entries[1].Action != ActionRestore {
		t.Fatalf("expected archive+restore audit entries, got %+v", audit.entries)
	}
}

func TestHardDelete_MustBeArchived(t *testing.T) {
	policy := &ArchivePolicy{ResourceType: "widget", AllowArchive: true, AllowHardDelete: true}
	svc, repo, _ := newTestService(t, policy)
	id := uuid.New()
	repo.seed(id) // not archived

	err := svc.HardDelete(context.Background(), &HardDeleteRequest{
		ResourceType:   "widget",
		ResourceID:     id,
		TenantID:       "argos-local-org",
		MustBeArchived: true,
	})
	if !errors.Is(err, ErrArchiveNotAllowed) {
		t.Fatalf("expected ErrArchiveNotAllowed, got %v", err)
	}
	if _, ok := repo.rows[id]; !ok {
		t.Fatalf("row should remain (not hard-deleted)")
	}
}

func TestBulkArchive_PerIDOutcomes(t *testing.T) {
	policy := &ArchivePolicy{ResourceType: "widget", AllowArchive: true}
	svc, repo, _ := newTestService(t, policy)
	id1, id2 := uuid.New(), uuid.New()
	repo.seed(id1)
	// id2 not seeded → SoftDelete will return NotFound.

	res, err := svc.BulkArchive(context.Background(), "widget", "argos-local-org", "actor", "", []uuid.UUID{id1, id2})
	if err != nil {
		t.Fatalf("BulkArchive top-level: %v", err)
	}
	if len(res) != 2 {
		t.Fatalf("expected 2 results, got %d", len(res))
	}
	if res[0].Err != nil {
		t.Errorf("id1 should have succeeded, got %v", res[0].Err)
	}
	if res[1].Err == nil {
		t.Errorf("id2 should have failed (not found)")
	}
}

func TestStaticPolicyRegistry_UnknownResource(t *testing.T) {
	reg := NewStaticPolicyRegistry(&ArchivePolicy{ResourceType: "widget", AllowArchive: true})
	if _, err := reg.Get("unknown"); !errors.Is(err, ErrPolicyNotFound) {
		t.Fatalf("expected ErrPolicyNotFound, got %v", err)
	}
}
