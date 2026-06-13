package lifecycle

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/devpablocristo/platform/errors/go/domainerr"
	"github.com/google/uuid"
)

type fakeRepo struct {
	rows         map[uuid.UUID]LifecycleState
	purgeAfter   map[uuid.UUID]*time.Time
	lastTenantID string
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		rows:       make(map[uuid.UUID]LifecycleState),
		purgeAfter: make(map[uuid.UUID]*time.Time),
	}
}

func (r *fakeRepo) seed(id uuid.UUID) { r.rows[id] = StateActive }

func (r *fakeRepo) Archive(_ context.Context, tenantID string, id uuid.UUID, _ time.Time) error {
	r.lastTenantID = tenantID
	state, ok := r.rows[id]
	if !ok || state != StateActive {
		return domainerr.NotFoundf("fake", id.String())
	}
	r.rows[id] = StateArchived
	return nil
}

func (r *fakeRepo) Unarchive(_ context.Context, tenantID string, id uuid.UUID) error {
	r.lastTenantID = tenantID
	state, ok := r.rows[id]
	if !ok || state != StateArchived {
		return domainerr.NotFoundf("fake", id.String())
	}
	r.rows[id] = StateActive
	return nil
}

func (r *fakeRepo) Trash(_ context.Context, tenantID string, id uuid.UUID, _ time.Time, purgeAfter *time.Time) error {
	r.lastTenantID = tenantID
	state, ok := r.rows[id]
	if !ok || state == StateTrashed || state == StatePurged {
		return domainerr.NotFoundf("fake", id.String())
	}
	r.rows[id] = StateTrashed
	if purgeAfter != nil {
		t := *purgeAfter
		r.purgeAfter[id] = &t
	}
	return nil
}

func (r *fakeRepo) Restore(_ context.Context, tenantID string, id uuid.UUID) error {
	r.lastTenantID = tenantID
	state, ok := r.rows[id]
	if !ok || state != StateTrashed {
		return domainerr.NotFoundf("fake", id.String())
	}
	r.rows[id] = StateActive
	delete(r.purgeAfter, id)
	return nil
}

func (r *fakeRepo) Purge(_ context.Context, tenantID string, id uuid.UUID) error {
	r.lastTenantID = tenantID
	if _, ok := r.rows[id]; !ok {
		return domainerr.NotFoundf("fake", id.String())
	}
	r.rows[id] = StatePurged
	delete(r.purgeAfter, id)
	return nil
}

func (r *fakeRepo) State(_ context.Context, tenantID string, id uuid.UUID) (LifecycleState, error) {
	r.lastTenantID = tenantID
	state, ok := r.rows[id]
	if !ok || state == StatePurged {
		return "", domainerr.NotFoundf("fake", id.String())
	}
	return state, nil
}

type recordingAudit struct{ entries []LifecycleAudit }

func (a *recordingAudit) Append(_ context.Context, e LifecycleAudit) error {
	a.entries = append(a.entries, e)
	return nil
}

func newTestService(t *testing.T, policies ...*LifecyclePolicy) (*Service, *fakeRepo, *recordingAudit) {
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

func TestArchive_HappyPath(t *testing.T) {
	policy := &LifecyclePolicy{ResourceType: "widget", AllowArchive: true}
	svc, repo, audit := newTestService(t, policy)
	id := uuid.New()
	repo.seed(id)

	err := svc.Archive(context.Background(), &ArchiveRequest{
		ResourceType: "widget",
		ResourceID:   id,
		TenantID:     "argos-local-org",
		Actor:        "tester@example.com",
		Reason:       "not active anymore",
	})
	if err != nil {
		t.Fatalf("Archive: %v", err)
	}
	if got := repo.rows[id]; got != StateArchived {
		t.Fatalf("expected archived state, got %q", got)
	}
	if repo.lastTenantID != "argos-local-org" {
		t.Fatalf("expected opaque tenant string, got %q", repo.lastTenantID)
	}
	if ParticipatesInAutomation(repo.rows[id]) {
		t.Fatal("archived resources should not participate in automation")
	}
	if len(audit.entries) != 1 || audit.entries[0].Action != ActionArchive {
		t.Fatalf("expected 1 archive audit entry, got %+v", audit.entries)
	}
	if audit.entries[0].FromState != StateActive || audit.entries[0].ToState != StateArchived {
		t.Fatalf("unexpected audit state transition: %+v", audit.entries[0])
	}
}

func TestArchive_RequiresReason(t *testing.T) {
	policy := &LifecyclePolicy{ResourceType: "widget", AllowArchive: true, RequireReason: true}
	svc, repo, _ := newTestService(t, policy)
	id := uuid.New()
	repo.seed(id)

	err := svc.Archive(context.Background(), &ArchiveRequest{
		ResourceType: "widget",
		ResourceID:   id,
		TenantID:     "argos-local-org",
	})
	if !errors.Is(err, ErrReasonRequired) {
		t.Fatalf("expected ErrReasonRequired, got %v", err)
	}
}

func TestArchive_BlockedByValidator(t *testing.T) {
	sentinel := errors.New("has active relations")
	policy := &LifecyclePolicy{
		ResourceType: "widget",
		AllowArchive: true,
		ValidateArchive: func(_ context.Context, _ string, _ uuid.UUID) error {
			return sentinel
		},
	}
	svc, repo, audit := newTestService(t, policy)
	id := uuid.New()
	repo.seed(id)

	err := svc.Archive(context.Background(), &ArchiveRequest{
		ResourceType: "widget",
		ResourceID:   id,
		TenantID:     "argos-local-org",
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected sentinel error, got %v", err)
	}
	if repo.rows[id] != StateActive {
		t.Fatalf("row should remain active")
	}
	if len(audit.entries) != 0 {
		t.Fatalf("no audit should be appended on rejection")
	}
}

func TestUnarchive_HappyPath(t *testing.T) {
	policy := &LifecyclePolicy{ResourceType: "widget", AllowArchive: true}
	svc, repo, audit := newTestService(t, policy)
	id := uuid.New()
	repo.seed(id)
	if err := svc.Archive(context.Background(), &ArchiveRequest{
		ResourceType: "widget",
		ResourceID:   id,
		TenantID:     "argos-local-org",
	}); err != nil {
		t.Fatalf("Archive: %v", err)
	}

	if err := svc.Unarchive(context.Background(), &UnarchiveRequest{
		ResourceType: "widget",
		ResourceID:   id,
		TenantID:     "argos-local-org",
		Actor:        "tester@example.com",
	}); err != nil {
		t.Fatalf("Unarchive: %v", err)
	}
	if got := repo.rows[id]; got != StateActive {
		t.Fatalf("expected active state, got %q", got)
	}
	if len(audit.entries) != 2 || audit.entries[1].Action != ActionUnarchive {
		t.Fatalf("expected archive+unarchive audit entries, got %+v", audit.entries)
	}
}

func TestTrashRestoreAndPurge(t *testing.T) {
	policy := &LifecyclePolicy{
		ResourceType:  "widget",
		AllowTrash:    true,
		AllowPurge:    true,
		RetentionDays: 30,
	}
	svc, repo, audit := newTestService(t, policy)
	id := uuid.New()
	repo.seed(id)

	if err := svc.Trash(context.Background(), &TrashRequest{
		ResourceType: "widget",
		ResourceID:   id,
		TenantID:     "argos-local-org",
		Actor:        "tester@example.com",
	}); err != nil {
		t.Fatalf("Trash: %v", err)
	}
	if got := repo.rows[id]; got != StateTrashed {
		t.Fatalf("expected trashed state, got %q", got)
	}
	if repo.purgeAfter[id] == nil {
		t.Fatalf("expected purgeAfter to be set")
	}
	if audit.entries[0].Action != ActionTrash || audit.entries[0].RetentionExpires == nil {
		t.Fatalf("expected trash audit with retention, got %+v", audit.entries[0])
	}

	if err := svc.Restore(context.Background(), &RestoreRequest{
		ResourceType: "widget",
		ResourceID:   id,
		TenantID:     "argos-local-org",
		Actor:        "tester@example.com",
	}); err != nil {
		t.Fatalf("Restore: %v", err)
	}
	if got := repo.rows[id]; got != StateActive {
		t.Fatalf("expected active state, got %q", got)
	}
	if repo.purgeAfter[id] != nil {
		t.Fatalf("expected purgeAfter cleared")
	}

	if err := svc.Trash(context.Background(), &TrashRequest{
		ResourceType: "widget",
		ResourceID:   id,
		TenantID:     "argos-local-org",
		Actor:        "tester@example.com",
	}); err != nil {
		t.Fatalf("Trash again: %v", err)
	}
	if err := svc.Purge(context.Background(), &PurgeRequest{
		ResourceType:  "widget",
		ResourceID:    id,
		TenantID:      "argos-local-org",
		Actor:         "tester@example.com",
		MustBeTrashed: true,
	}); err != nil {
		t.Fatalf("Purge: %v", err)
	}
	if got := repo.rows[id]; got != StatePurged {
		t.Fatalf("expected purged state, got %q", got)
	}
}

func TestPurgeMustBeTrashed(t *testing.T) {
	policy := &LifecyclePolicy{ResourceType: "widget", AllowPurge: true}
	svc, repo, _ := newTestService(t, policy)
	id := uuid.New()
	repo.seed(id)

	err := svc.Purge(context.Background(), &PurgeRequest{
		ResourceType:  "widget",
		ResourceID:    id,
		TenantID:      "argos-local-org",
		MustBeTrashed: true,
	})
	if !errors.Is(err, ErrMustBeTrashed) {
		t.Fatalf("expected ErrMustBeTrashed, got %v", err)
	}
	if got := repo.rows[id]; got != StateActive {
		t.Fatalf("row should remain active, got %q", got)
	}
}

func TestBulkArchive_PerIDOutcomes(t *testing.T) {
	policy := &LifecyclePolicy{ResourceType: "widget", AllowArchive: true}
	svc, repo, _ := newTestService(t, policy)
	id1, id2 := uuid.New(), uuid.New()
	repo.seed(id1)

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
	reg := NewStaticPolicyRegistry(&LifecyclePolicy{ResourceType: "widget", AllowArchive: true})
	if _, err := reg.Get("unknown"); !errors.Is(err, ErrPolicyNotFound) {
		t.Fatalf("expected ErrPolicyNotFound, got %v", err)
	}
}
