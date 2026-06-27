package dto

import billingdomain "github.com/devpablocristo/platform/kernels/saas/go/billing/usecases/domain"

type OrgRequest struct {
	OrgID    string `json:"org_id"`
	TenantID string `json:"-"`
}

type OrgBillingResponse struct {
	Billing billingdomain.OrgBilling `json:"billing"`
}

type TenantRequest = OrgRequest
type TenantBillingResponse = OrgBillingResponse

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
	OrgID    string                      `json:"org_id"`
	TenantID string                      `json:"-"`
	PlanCode billingdomain.PlanCode      `json:"plan_code"`
	Status   billingdomain.BillingStatus `json:"status"`
	Actor    *string                     `json:"actor,omitempty"`
}
