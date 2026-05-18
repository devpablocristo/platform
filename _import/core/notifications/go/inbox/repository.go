package inbox

import (
	"context"
	"time"

	domain "github.com/devpablocristo/core/notifications/go/inbox/usecases/domain"
)

// Repository define persistencia para la bandeja in-app.
type Repository interface {
	ListForRecipient(ctx context.Context, orgID, recipientID string, limit int) ([]domain.Notification, error)
	CountUnread(ctx context.Context, orgID, recipientID string) (int64, error)
	Append(ctx context.Context, notification domain.Notification) (domain.Notification, error)
	MarkRead(ctx context.Context, orgID, recipientID, notificationID string, readAt time.Time) (time.Time, error)
}
