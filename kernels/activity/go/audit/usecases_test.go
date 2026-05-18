package audit

import (
	"context"
	"strings"
	"testing"
	"time"

	domain "github.com/devpablocristo/core/activity/go/audit/usecases/domain"
	kernel "github.com/devpablocristo/core/activity/go/kernel/usecases/domain"
)

func TestAppendBuildsHashChain(t *testing.T) {
	t.Parallel()

	repo := &fakeAuditRepository{lastHash: "prev"}
	usecases := NewUsecases(repo)
	usecases.now = func() time.Time { return time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC) }

	entry, err := usecases.Append(context.Background(), domain.LogInput{
		TenantID:     "acme",
		Actor:        kernel.ActorRef{Legacy: "pablo", Type: "user"},
		Action:       "invoice.created",
		ResourceType: "invoice",
		ResourceID:   "inv_1",
		Payload:      map[string]any{"amount": 10},
	})
	if err != nil {
		t.Fatalf("Append returned error: %v", err)
	}
	if entry.PrevHash != "prev" {
		t.Fatalf("unexpected prev hash: %q", entry.PrevHash)
	}
	if entry.Hash == "" {
		t.Fatal("expected hash")
	}
}

func TestExportCSVAndJSONL(t *testing.T) {
	t.Parallel()

	repo := &fakeAuditRepository{
		items: []domain.Entry{{
			ID:           "evt_1",
			TenantID:     "acme",
			Actor:        "pablo",
			ActorType:    "user",
			ActorLabel:   "Pablo",
			Action:       "invoice.created",
			ResourceType: "invoice",
			ResourceID:   "inv_1",
			Hash:         "abc",
			CreatedAt:    time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC),
			Payload:      map[string]any{"amount": 10},
		}},
	}
	usecases := NewUsecases(repo)

	csvBody, err := usecases.ExportCSV(context.Background(), "acme", 10)
	if err != nil {
		t.Fatalf("ExportCSV returned error: %v", err)
	}
	if !strings.Contains(csvBody, "invoice.created") {
		t.Fatalf("expected action in CSV: %q", csvBody)
	}

	jsonl, err := usecases.ExportJSONL(context.Background(), "acme", 10)
	if err != nil {
		t.Fatalf("ExportJSONL returned error: %v", err)
	}
	if !strings.Contains(jsonl, "\"action\":\"invoice.created\"") {
		t.Fatalf("expected action in JSONL: %q", jsonl)
	}
}

type fakeAuditRepository struct {
	lastHash string
	items    []domain.Entry
	appended domain.Entry
}

func (r *fakeAuditRepository) LastHash(_ context.Context, _ string) (string, error) {
	return r.lastHash, nil
}

func (r *fakeAuditRepository) Append(_ context.Context, entry domain.Entry) (domain.Entry, error) {
	r.appended = entry
	return entry, nil
}

func (r *fakeAuditRepository) List(_ context.Context, _ string, _ int) ([]domain.Entry, error) {
	return append([]domain.Entry(nil), r.items...), nil
}
