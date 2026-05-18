package billing

import (
	"context"

	billingdomain "github.com/devpablocristo/core/saas/go/billing/usecases/domain"
)

// Repository define el puerto de persistencia del contexto billing.
type Repository interface {
	GetTenantBilling(context.Context, string) (billingdomain.TenantBilling, bool, error)
	UpsertTenantBilling(context.Context, billingdomain.TenantBilling) (billingdomain.TenantBilling, error)
	GetUsageSummary(context.Context, string) (billingdomain.UsageSummary, error)
}
