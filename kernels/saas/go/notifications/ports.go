package notifications

import "context"

// NotificationPort define el envío de notificaciones SaaS agnósticas del canal.
type NotificationPort interface {
	Notify(ctx context.Context, tenantID, notificationType string, data map[string]string) error
}
