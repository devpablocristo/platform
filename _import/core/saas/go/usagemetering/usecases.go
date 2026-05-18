package usagemetering

import (
	"context"
	"strings"
	"time"

	usage "github.com/devpablocristo/core/saas/go/usagemetering/usecases/domain"
)

type UseCases struct {
	metering MeteringPort
}

func NewUseCases(metering MeteringPort) *UseCases {
	return &UseCases{metering: metering}
}

func (u *UseCases) Increment(ctx context.Context, tenantID string, counter usage.Counter) error {
	return u.metering.Increment(ctx, strings.TrimSpace(tenantID), string(counter))
}

func CurrentPeriod(now time.Time) string {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return now.UTC().Format("2006-01")
}
