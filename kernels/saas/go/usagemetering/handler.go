package usagemetering

import (
	"context"
	"time"

	"github.com/devpablocristo/core/saas/go/usagemetering/handler/dto"
)

// Handler expone un adapter de aplicación listo para transporte externo.
type Handler struct {
	usecases *UseCases
}

func NewHandler(usecases *UseCases) *Handler {
	return &Handler{usecases: usecases}
}

func (h *Handler) Increment(ctx context.Context, input dto.IncrementRequest) error {
	return h.usecases.Increment(ctx, input.TenantID, input.Counter)
}

func (h *Handler) CurrentPeriod(input dto.CurrentPeriodRequest) dto.CurrentPeriodResponse {
	now := input.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return dto.CurrentPeriodResponse{Period: CurrentPeriod(now)}
}
