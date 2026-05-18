package usagemetering

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/devpablocristo/core/saas/go/middleware"
	usage "github.com/devpablocristo/core/saas/go/usagemetering/usecases/domain"
)

const maxConcurrentMetering = 512

func NewAPICallsMiddleware(m MeteringPort) func(http.Handler) http.Handler {
	sem := make(chan struct{}, maxConcurrentMetering)
	return func(next http.Handler) http.Handler {
		if next == nil {
			next = http.NotFoundHandler()
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)

			principal, ok := middleware.PrincipalFromContext(r.Context())
			if !ok || principal.TenantID == "" {
				return
			}

			select {
			case sem <- struct{}{}:
				go func(tenantID string) {
					defer func() { <-sem }()
					ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
					defer cancel()
					if err := m.Increment(ctx, tenantID, string(usage.CounterAPICalls)); err != nil {
						slog.Warn("usage metering increment failed", "tenant_id", tenantID, "error", err)
					}
				}(principal.TenantID)
			default:
				slog.Warn("usage metering concurrency limit reached", "tenant_id", principal.TenantID)
			}
		})
	}
}
