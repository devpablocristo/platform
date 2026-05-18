package candidates

import (
	"context"
	"testing"
	"time"

	domain "github.com/devpablocristo/core/notifications/go/candidates/usecases/domain"
)

type repositoryStub struct {
	lastRecordOrg string
	lastRecordInput  domain.UpsertInput
	lastMarkTenant   string
	lastMarkID       string
	lastListTenant   string
	lastListLimit    int
}

func (s *repositoryStub) Upsert(_ context.Context, in domain.UpsertInput) (domain.Candidate, bool, error) {
	s.lastRecordOrg = in.OrgID
	s.lastRecordInput = in
	return domain.Candidate{ID: "cand-1", OrgID: in.OrgID, Title: in.Title}, true, nil
}

func (s *repositoryStub) MarkNotified(_ context.Context, orgID, candidateID string, _ time.Time) error {
	s.lastMarkTenant = orgID
	s.lastMarkID = candidateID
	return nil
}

func (s *repositoryStub) ListByTenant(_ context.Context, orgID string, limit int) ([]domain.Candidate, error) {
	s.lastListTenant = orgID
	s.lastListLimit = limit
	return []domain.Candidate{{ID: "cand-1", OrgID: orgID}}, nil
}

func TestRecordValidatesRequiredFieldsAndDefaults(t *testing.T) {
	t.Parallel()

	repo := &repositoryStub{}
	uc := NewUsecases(repo)

	record, shouldNotify, err := uc.Record(context.Background(), domain.UpsertInput{
		OrgID:    "tenant-1",
		EventType:   "sale.created",
		EntityType:  "sale",
		EntityID:    "sale-1",
		Fingerprint: "sale.created:sale-1",
		Title:       "Venta destacada",
		Body:        "Se registró una venta destacada.",
	})
	if err != nil {
		t.Fatalf("Record() error = %v", err)
	}
	if !shouldNotify {
		t.Fatalf("expected shouldNotify=true")
	}
	if record.ID == "" {
		t.Fatalf("expected candidate id")
	}
	if repo.lastRecordInput.Kind != "insight" {
		t.Fatalf("expected default kind insight, got %q", repo.lastRecordInput.Kind)
	}
	if repo.lastRecordInput.Severity != "info" {
		t.Fatalf("expected default severity info, got %q", repo.lastRecordInput.Severity)
	}
	if repo.lastRecordInput.Now.IsZero() {
		t.Fatalf("expected now to be populated")
	}
}

func TestMarkNotifiedDelegatesToRepository(t *testing.T) {
	t.Parallel()

	repo := &repositoryStub{}
	uc := NewUsecases(repo)
	if err := uc.MarkNotified(context.Background(), "tenant-1", "cand-1"); err != nil {
		t.Fatalf("MarkNotified() error = %v", err)
	}
	if repo.lastMarkTenant != "tenant-1" || repo.lastMarkID != "cand-1" {
		t.Fatalf("unexpected mark call tenant=%q candidate=%q", repo.lastMarkTenant, repo.lastMarkID)
	}
}

func TestListUsesBoundedLimit(t *testing.T) {
	t.Parallel()

	repo := &repositoryStub{}
	uc := NewUsecases(repo)
	items, err := uc.List(context.Background(), "tenant-1", 1000)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if repo.lastListTenant != "tenant-1" || repo.lastListLimit != 100 {
		t.Fatalf("unexpected list call tenant=%q limit=%d", repo.lastListTenant, repo.lastListLimit)
	}
}
