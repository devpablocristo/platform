package timeline

import (
	"context"
	"testing"
	"time"

	domain "github.com/devpablocristo/core/activity/go/timeline/usecases/domain"
)

func TestRecordNormalizesEntry(t *testing.T) {
	t.Parallel()

	repo := &fakeTimelineRepository{}
	usecases := NewUsecases(repo)
	usecases.now = func() time.Time { return time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC) }

	entry, err := usecases.Record(context.Background(), domain.Entry{
		TenantID:   "acme",
		EntityType: "invoice",
		EntityID:   "inv_1",
		EventType:  "invoice.sent",
		Title:      " Invoice sent ",
	})
	if err != nil {
		t.Fatalf("Record returned error: %v", err)
	}
	if entry.ID == "" {
		t.Fatal("expected generated id")
	}
	if entry.Title != "Invoice sent" {
		t.Fatalf("unexpected title: %q", entry.Title)
	}
}

type fakeTimelineRepository struct {
	last domain.Entry
}

func (r *fakeTimelineRepository) List(_ context.Context, _, _, _ string, _ int) ([]domain.Entry, error) {
	return []domain.Entry{r.last}, nil
}

func (r *fakeTimelineRepository) Append(_ context.Context, entry domain.Entry) (domain.Entry, error) {
	r.last = entry
	return entry, nil
}
