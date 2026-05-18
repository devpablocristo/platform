package identity

import "github.com/devpablocristo/core/saas/go/identity/handler/dto"

// Handler expone un adapter de aplicación listo para transporte externo.
type Handler struct {
	usecases *UseCases
}

func NewHandler(usecases *UseCases) *Handler {
	return &Handler{usecases: usecases}
}

func (h *Handler) Verify(input dto.VerifyRequest) (dto.VerifyResponse, error) {
	principal, err := h.usecases.Verify(input.Token)
	if err != nil {
		return dto.VerifyResponse{}, err
	}
	return dto.VerifyResponse{Principal: principal}, nil
}

func (h *Handler) BearerToken(input dto.TokenHeaderRequest) dto.TokenHeaderResponse {
	token, ok := h.usecases.BearerToken(input.Authorization)
	return dto.TokenHeaderResponse{Token: token, Found: ok}
}

func (h *Handler) APIKeyToken(input dto.APIKeyHeaderRequest) dto.TokenHeaderResponse {
	token, ok := h.usecases.APIKeyToken(input.Authorization, input.XAPIKey)
	return dto.TokenHeaderResponse{Token: token, Found: ok}
}
