package inbox

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	domain "github.com/devpablocristo/core/notifications/go/inbox/usecases/domain"
)

type Usecases struct {
	repo Repository
	now  func() time.Time
}

func NewUsecases(repo Repository) *Usecases {
	return &Usecases{
		repo: repo,
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

func (u *Usecases) Create(ctx context.Context, notification domain.Notification) (domain.Notification, error) {
	if strings.TrimSpace(notification.OrgID) == "" {
		return domain.Notification{}, fmt.Errorf("org_id is required")
	}
	if strings.TrimSpace(notification.RecipientID) == "" {
		return domain.Notification{}, fmt.Errorf("recipient_id is required")
	}
	if strings.TrimSpace(notification.Title) == "" {
		return domain.Notification{}, fmt.Errorf("title is required")
	}
	if strings.TrimSpace(notification.Body) == "" {
		return domain.Notification{}, fmt.Errorf("body is required")
	}

	notification.ID = firstNonEmpty(notification.ID, newID())
	notification.OrgID = strings.TrimSpace(notification.OrgID)
	notification.RecipientID = strings.TrimSpace(notification.RecipientID)
	notification.Title = strings.TrimSpace(notification.Title)
	notification.Body = strings.TrimSpace(notification.Body)
	notification.Kind = firstNonEmpty(strings.TrimSpace(notification.Kind), "system")
	notification.EntityType = strings.TrimSpace(notification.EntityType)
	notification.EntityID = strings.TrimSpace(notification.EntityID)
	notification.Metadata = normalizeMetadata(notification.Metadata)
	notification.ReadAt = nil
	if notification.CreatedAt.IsZero() {
		notification.CreatedAt = u.now().UTC()
	} else {
		notification.CreatedAt = notification.CreatedAt.UTC()
	}

	return u.repo.Append(ctx, notification)
}

func (u *Usecases) ListForRecipient(ctx context.Context, orgID, recipientID string, limit int) ([]domain.Notification, error) {
	orgID = strings.TrimSpace(orgID)
	if orgID == "" {
		return nil, fmt.Errorf("org_id is required")
	}
	recipientID = strings.TrimSpace(recipientID)
	if recipientID == "" {
		return nil, fmt.Errorf("recipient_id is required")
	}
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	return u.repo.ListForRecipient(ctx, orgID, recipientID, limit)
}

func (u *Usecases) CountUnread(ctx context.Context, orgID, recipientID string) (int64, error) {
	orgID = strings.TrimSpace(orgID)
	if orgID == "" {
		return 0, fmt.Errorf("org_id is required")
	}
	recipientID = strings.TrimSpace(recipientID)
	if recipientID == "" {
		return 0, fmt.Errorf("recipient_id is required")
	}
	return u.repo.CountUnread(ctx, orgID, recipientID)
}

func (u *Usecases) MarkRead(ctx context.Context, orgID, recipientID, notificationID string) (time.Time, error) {
	orgID = strings.TrimSpace(orgID)
	if orgID == "" {
		return time.Time{}, fmt.Errorf("org_id is required")
	}
	recipientID = strings.TrimSpace(recipientID)
	if recipientID == "" {
		return time.Time{}, fmt.Errorf("recipient_id is required")
	}
	notificationID = strings.TrimSpace(notificationID)
	if notificationID == "" {
		return time.Time{}, fmt.Errorf("notification_id is required")
	}
	return u.repo.MarkRead(ctx, orgID, recipientID, notificationID, u.now().UTC())
}

func normalizeMetadata(raw json.RawMessage) json.RawMessage {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		return json.RawMessage(`{}`)
	}
	return json.RawMessage(trimmed)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func newID() string {
	var data [16]byte
	if _, err := rand.Read(data[:]); err != nil {
		return fmt.Sprintf("notification-%d", time.Now().UTC().UnixNano())
	}
	return hex.EncodeToString(data[:])
}
