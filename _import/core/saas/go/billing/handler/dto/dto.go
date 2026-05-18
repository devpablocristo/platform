package dto

import billingdomain "github.com/devpablocristo/core/saas/go/billing/usecases/domain"

type TenantRequest struct {
	TenantID string `json:"tenant_id"`
}

type TenantBillingResponse struct {
	Billing billingdomain.TenantBilling `json:"billing"`
}

type BillingStatusResponse struct {
	Status billingdomain.BillingStatusView `json:"status"`
}

type CheckoutRequest struct {
	Checkout billingdomain.CheckoutInput `json:"checkout"`
}

type PortalRequest struct {
	Portal billingdomain.PortalInput `json:"portal"`
}

type SessionResponse struct {
	URL string `json:"url"`
}

type ApplyPlanChangeRequest struct {
	TenantID string                      `json:"tenant_id"`
	PlanCode billingdomain.PlanCode      `json:"plan_code"`
	Status   billingdomain.BillingStatus `json:"status"`
	Actor    *string                     `json:"actor,omitempty"`
}
