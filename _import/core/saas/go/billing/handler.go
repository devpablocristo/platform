package billing

import (
	"context"

	"github.com/devpablocristo/core/saas/go/billing/handler/dto"
)

// Handler expone un adapter de aplicación listo para transporte externo.
type Handler struct {
	usecases *UseCases
}

func NewHandler(usecases *UseCases) *Handler {
	return &Handler{usecases: usecases}
}

func (h *Handler) EnsureTenantBilling(ctx context.Context, input dto.TenantRequest) (dto.TenantBillingResponse, error) {
	item, err := h.usecases.EnsureTenantBilling(ctx, input.TenantID)
	if err != nil {
		return dto.TenantBillingResponse{}, err
	}
	return dto.TenantBillingResponse{Billing: item}, nil
}

func (h *Handler) GetBillingStatus(ctx context.Context, input dto.TenantRequest) (dto.BillingStatusResponse, error) {
	view, err := h.usecases.GetBillingStatus(ctx, input.TenantID)
	if err != nil {
		return dto.BillingStatusResponse{}, err
	}
	return dto.BillingStatusResponse{Status: view}, nil
}

func (h *Handler) CreateCheckoutSession(ctx context.Context, input dto.CheckoutRequest) (dto.SessionResponse, error) {
	url, err := h.usecases.CreateCheckoutSession(ctx, input.Checkout)
	if err != nil {
		return dto.SessionResponse{}, err
	}
	return dto.SessionResponse{URL: url}, nil
}

func (h *Handler) CreatePortalSession(ctx context.Context, input dto.PortalRequest) (dto.SessionResponse, error) {
	url, err := h.usecases.CreatePortalSession(ctx, input.Portal)
	if err != nil {
		return dto.SessionResponse{}, err
	}
	return dto.SessionResponse{URL: url}, nil
}

func (h *Handler) ApplyPlanChange(ctx context.Context, input dto.ApplyPlanChangeRequest) (dto.TenantBillingResponse, error) {
	item, err := h.usecases.ApplyPlanChange(ctx, input.TenantID, input.PlanCode, input.Status, input.Actor)
	if err != nil {
		return dto.TenantBillingResponse{}, err
	}
	return dto.TenantBillingResponse{Billing: item}, nil
}
