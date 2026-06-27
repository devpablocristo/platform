package admin

import (
	"context"

	"github.com/devpablocristo/platform/kernels/saas/go/admin/handler/dto"
)

// Handler expone un adapter de aplicación listo para transporte externo.
type Handler struct {
	usecases *UseCases
}

func NewHandler(usecases *UseCases) *Handler {
	return &Handler{usecases: usecases}
}

func (h *Handler) GetOrgSettings(ctx context.Context, input dto.GetOrgSettingsRequest) (dto.OrgSettingsResponse, error) {
	orgID := firstNonEmpty(input.OrgID, input.TenantID)
	item, err := h.usecases.GetOrgSettings(ctx, orgID, input.Actor, input.Role, input.Scopes)
	if err != nil {
		return dto.OrgSettingsResponse{}, err
	}
	return dto.OrgSettingsResponse{Settings: item}, nil
}

func (h *Handler) UpsertOrgSettings(ctx context.Context, input dto.UpsertOrgSettingsRequest) (dto.OrgSettingsResponse, error) {
	orgID := firstNonEmpty(input.OrgID, input.TenantID)
	item, err := h.usecases.UpsertOrgSettings(ctx, orgID, input.Actor, input.Role, input.Scopes, input.PlanCode, input.HardLimits)
	if err != nil {
		return dto.OrgSettingsResponse{}, err
	}
	return dto.OrgSettingsResponse{Settings: item}, nil
}

func (h *Handler) UpdateLifecycle(ctx context.Context, input dto.UpdateLifecycleRequest) (dto.OrgSettingsResponse, error) {
	orgID := firstNonEmpty(input.OrgID, input.TenantID)
	item, err := h.usecases.UpdateLifecycle(ctx, orgID, input.Actor, input.Role, input.Scopes, input.Status)
	if err != nil {
		return dto.OrgSettingsResponse{}, err
	}
	return dto.OrgSettingsResponse{Settings: item}, nil
}

func (h *Handler) GetTenantSettings(ctx context.Context, input dto.GetTenantSettingsRequest) (dto.TenantSettingsResponse, error) {
	return h.GetOrgSettings(ctx, input)
}

func (h *Handler) UpsertTenantSettings(ctx context.Context, input dto.UpsertTenantSettingsRequest) (dto.TenantSettingsResponse, error) {
	return h.UpsertOrgSettings(ctx, input)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
