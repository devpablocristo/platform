package admin

import (
	"context"

	"github.com/devpablocristo/core/saas/go/admin/handler/dto"
)

// Handler expone un adapter de aplicación listo para transporte externo.
type Handler struct {
	usecases *UseCases
}

func NewHandler(usecases *UseCases) *Handler {
	return &Handler{usecases: usecases}
}

func (h *Handler) GetTenantSettings(ctx context.Context, input dto.GetTenantSettingsRequest) (dto.TenantSettingsResponse, error) {
	item, err := h.usecases.GetTenantSettings(ctx, input.TenantID, input.Actor, input.Role, input.Scopes)
	if err != nil {
		return dto.TenantSettingsResponse{}, err
	}
	return dto.TenantSettingsResponse{Settings: item}, nil
}

func (h *Handler) UpsertTenantSettings(ctx context.Context, input dto.UpsertTenantSettingsRequest) (dto.TenantSettingsResponse, error) {
	item, err := h.usecases.UpsertTenantSettings(ctx, input.TenantID, input.Actor, input.Role, input.Scopes, input.PlanCode, input.HardLimits)
	if err != nil {
		return dto.TenantSettingsResponse{}, err
	}
	return dto.TenantSettingsResponse{Settings: item}, nil
}

func (h *Handler) UpdateLifecycle(ctx context.Context, input dto.UpdateLifecycleRequest) (dto.TenantSettingsResponse, error) {
	item, err := h.usecases.UpdateLifecycle(ctx, input.TenantID, input.Actor, input.Role, input.Scopes, input.Status)
	if err != nil {
		return dto.TenantSettingsResponse{}, err
	}
	return dto.TenantSettingsResponse{Settings: item}, nil
}
