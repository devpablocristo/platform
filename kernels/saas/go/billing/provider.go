package billing

import (
	"context"

	billingdomain "github.com/devpablocristo/core/saas/go/billing/usecases/domain"
)

// CheckoutProvider define el puerto del proveedor de checkout/portal.
type CheckoutProvider interface {
	CreateCheckoutSession(context.Context, billingdomain.CheckoutInput, billingdomain.TenantBilling) (string, error)
	CreatePortalSession(context.Context, billingdomain.PortalInput, billingdomain.TenantBilling) (string, error)
}
