package org

import (
	"context"

	"github.com/devpablocristo/core/saas/go/org/handler/dto"
)

// Handler expone un adapter de aplicación listo para transporte externo.
type Handler struct {
	usecases *UseCases
}

func NewHandler(usecases *UseCases) *Handler {
	return &Handler{usecases: usecases}
}

func (h *Handler) ResolvePrincipal(ctx context.Context, input dto.ResolvePrincipalRequest) (dto.ResolvePrincipalResponse, error) {
	principal, err := h.usecases.ResolvePrincipal(ctx, input.APIKeyHash)
	if err != nil {
		return dto.ResolvePrincipalResponse{}, err
	}
	return dto.ResolvePrincipalResponse{Principal: principal}, nil
}
