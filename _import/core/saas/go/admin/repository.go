package admin

import (
	"context"
	"time"

	admindomain "github.com/devpablocristo/core/saas/go/admin/usecases/domain"
)

// Repository define el puerto de persistencia del contexto admin.
type Repository interface {
	GetTenantSettings(context.Context, string) (admindomain.TenantSettings, bool, error)
	UpsertTenantSettings(context.Context, admindomain.TenantSettings) (admindomain.TenantSettings, error)
	UpdateTenantLifecycle(context.Context, string, admindomain.TenantStatus, *time.Time, *string) (admindomain.TenantSettings, error)
	CreateAdminActivityEvent(context.Context, admindomain.AdminActivityEvent) error
	ListAdminActivityEvents(context.Context, string, int) ([]admindomain.AdminActivityEvent, error)
}
