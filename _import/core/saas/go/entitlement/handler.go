package entitlement

import "github.com/devpablocristo/core/saas/go/entitlement/handler/dto"

// Handler expone un adapter de aplicación listo para transporte externo.
type Handler struct {
	usecases *UseCases
}

func NewHandler(usecases *UseCases) *Handler {
	return &Handler{usecases: usecases}
}

func (h *Handler) NormalizePlan(input dto.NormalizePlanRequest) dto.NormalizePlanResponse {
	return dto.NormalizePlanResponse{Plan: h.usecases.NormalizePlan(string(input.Plan))}
}

func (h *Handler) DefaultHardLimits(input dto.NormalizePlanRequest) dto.DefaultHardLimitsResponse {
	return dto.DefaultHardLimitsResponse{Limits: h.usecases.DefaultHardLimits(input.Plan)}
}

func (h *Handler) CanUse(input dto.CanUseRequest) dto.BoolResponse {
	return dto.BoolResponse{Allowed: h.usecases.CanUse(input.Current, input.Limit)}
}
