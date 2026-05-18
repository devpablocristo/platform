package authz

import "github.com/devpablocristo/core/authz/go/handler/dto"

// Handler expone un adapter de aplicación listo para transporte externo.
type Handler struct {
	usecases *UseCases
}

func NewHandler(usecases *UseCases) *Handler {
	return &Handler{usecases: usecases}
}

func (h *Handler) HasScope(input dto.HasScopeRequest) dto.BoolResponse {
	return dto.BoolResponse{Allowed: h.usecases.HasScope(input.Scopes, input.Required)}
}

func (h *Handler) HasAnyScope(input dto.HasAnyScopeRequest) dto.BoolResponse {
	return dto.BoolResponse{Allowed: h.usecases.HasAnyScope(input.Scopes, input.Required...)}
}

func (h *Handler) IsRole(input dto.IsRoleRequest) dto.BoolResponse {
	return dto.BoolResponse{Allowed: h.usecases.IsRole(input.Role, input.Accepted...)}
}
