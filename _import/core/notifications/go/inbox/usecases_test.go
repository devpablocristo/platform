package inbox

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	domain "github.com/devpablocristo/core/notifications/go/inbox/usecases/domain"
)

type stubRepository struct {
	appended           domain.Notification
	listOrgID       string
	listRecipientID    string
	listLimit          int
	unreadOrgID     string
	unreadRecipientID  string
	markOrgID       string
	markRecipientID    string
	markNotificationID string
	markReadAt         time.Time
	listOut            []domain.Notification
	unreadOut          int64
}

func (s *stubRepository) ListForRecipient(_ context.Context, orgID, recipientID string, limit int) ([]domain.Notification, error) {
	s.listOrgID = orgID
	s.listRecipientID = recipientID
	s.listLimit = limit
	return s.listOut, nil
}

func (s *stubRepository) CountUnread(_ context.Context, orgID, recipientID string) (int64, error) {
	s.unreadOrgID = orgID
	s.unreadRecipientID = recipientID
	return s.unreadOut, nil
}

func (s *stubRepository) Append(_ context.Context, notification domain.Notification) (domain.Notification, error) {
	s.appended = notification
	return notification, nil
}

func (s *stubRepository) MarkRead(_ context.Context, orgID, recipientID, notificationID string, readAt time.Time) (time.Time, error) {
	s.markOrgID = orgID
	s.markRecipientID = recipientID
	s.markNotificationID = notificationID
	s.markReadAt = readAt
	return readAt, nil
}

func TestCreateAppliesDefaults(t *testing.T) {
	t.Parallel()

	repo := &stubRepository{}
	uc := NewUsecases(repo)
	fixedNow := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	uc.now = func() time.Time { return fixedNow }

	created, err := uc.Create(context.Background(), domain.Notification{
		OrgID:    " tenant-1 ",
		RecipientID: " user-1 ",
		Title:       "  Aviso  ",
		Body:        "  Cuerpo  ",
		Metadata:    nil,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if created.ID == "" {
		t.Fatal("expected generated id")
	}
	if created.Kind != "system" {
		t.Fatalf("expected default kind system, got %q", created.Kind)
	}
	if string(created.Metadata) != "{}" {
		t.Fatalf("expected default metadata, got %s", created.Metadata)
	}
	if !created.CreatedAt.Equal(fixedNow) {
		t.Fatalf("expected created_at %s, got %s", fixedNow, created.CreatedAt)
	}
	if repo.appended.OrgID != "tenant-1" || repo.appended.RecipientID != "user-1" {
		t.Fatalf("unexpected normalized notification: %+v", repo.appended)
	}
}

func TestCreateKeepsMetadataPayload(t *testing.T) {
	t.Parallel()

	repo := &stubRepository{}
	uc := NewUsecases(repo)
	in := json.RawMessage(`{"scope":"sales"}`)

	created, err := uc.Create(context.Background(), domain.Notification{
		OrgID:    "tenant-1",
		RecipientID: "user-1",
		Title:       "Aviso",
		Body:        "Cuerpo",
		Metadata:    in,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if string(created.Metadata) != string(in) {
		t.Fatalf("expected metadata %s, got %s", in, created.Metadata)
	}
}

func TestListForRecipientNormalizesLimit(t *testing.T) {
	t.Parallel()

	repo := &stubRepository{listOut: []domain.Notification{{ID: "n-1"}}}
	uc := NewUsecases(repo)

	items, err := uc.ListForRecipient(context.Background(), "tenant-1", "user-1", 0)
	if err != nil {
		t.Fatalf("ListForRecipient() error = %v", err)
	}

	if len(items) != 1 || items[0].ID != "n-1" {
		t.Fatalf("unexpected items: %+v", items)
	}
	if repo.listLimit != 100 {
		t.Fatalf("expected normalized limit 100, got %d", repo.listLimit)
	}
}

func TestCountUnreadDelegatesToRepository(t *testing.T) {
	t.Parallel()

	repo := &stubRepository{unreadOut: 7}
	uc := NewUsecases(repo)

	count, err := uc.CountUnread(context.Background(), "tenant-1", "user-1")
	if err != nil {
		t.Fatalf("CountUnread() error = %v", err)
	}

	if count != 7 {
		t.Fatalf("expected unread 7, got %d", count)
	}
	if repo.unreadOrgID != "tenant-1" || repo.unreadRecipientID != "user-1" {
		t.Fatalf("unexpected unread args: tenant=%q recipient=%q", repo.unreadOrgID, repo.unreadRecipientID)
	}
}

func TestMarkReadUsesClock(t *testing.T) {
	t.Parallel()

	repo := &stubRepository{}
	uc := NewUsecases(repo)
	fixedNow := time.Date(2026, 4, 1, 15, 30, 0, 0, time.UTC)
	uc.now = func() time.Time { return fixedNow }

	readAt, err := uc.MarkRead(context.Background(), "tenant-1", "user-1", "notif-1")
	if err != nil {
		t.Fatalf("MarkRead() error = %v", err)
	}

	if !readAt.Equal(fixedNow) {
		t.Fatalf("expected readAt %s, got %s", fixedNow, readAt)
	}
	if repo.markNotificationID != "notif-1" {
		t.Fatalf("expected notification id notif-1, got %q", repo.markNotificationID)
	}
}
