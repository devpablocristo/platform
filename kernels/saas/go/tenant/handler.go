package tenant

import "github.com/devpablocristo/core/saas/go/tenant/handler/dto"

// Handler expone un adapter de aplicación listo para transporte externo.
type Handler struct {
	usecases *UseCases
}

func NewHandler(usecases *UseCases) *Handler {
	return &Handler{usecases: usecases}
}

func (h *Handler) NormalizeSlug(input dto.NormalizeRequest) dto.StringResponse {
	return dto.StringResponse{Value: h.usecases.NormalizeSlug(input.Value)}
}

func (h *Handler) NormalizeRole(input dto.NormalizeRequest) dto.StringResponse {
	return dto.StringResponse{Value: h.usecases.NormalizeRole(input.Value)}
}

func (h *Handler) NewMembership(input dto.NewMembershipRequest) dto.NewMembershipResponse {
	return dto.NewMembershipResponse{
		Membership: h.usecases.NewMembership(input.TenantID, input.UserID, input.Role),
	}
}
