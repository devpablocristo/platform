package billing

import (
	"context"
	"log/slog"
	"strings"
	"time"
)

const defaultGracePeriod = 14 * 24 * time.Hour

type AutoSuspendPort interface {
	AutoSuspend(context.Context, string) error
}

type DunningWorker struct {
	repo        RuntimeRepository
	admin       AutoSuspendPort
	gracePeriod time.Duration
	logger      *slog.Logger
}

func NewDunningWorker(repo RuntimeRepository, admin AutoSuspendPort, logger *slog.Logger) *DunningWorker {
	if logger == nil {
		logger = slog.Default()
	}
	return &DunningWorker{
		repo:        repo,
		admin:       admin,
		gracePeriod: defaultGracePeriod,
		logger:      logger,
	}
}

func (w *DunningWorker) RunOnce(ctx context.Context) {
	if w == nil || w.repo == nil || w.admin == nil {
		return
	}

	cutoff := time.Now().UTC().Add(-w.gracePeriod)
	tenants, err := w.repo.FindPastDueBefore(ctx, cutoff)
	if err != nil {
		w.logger.Error("dunning: failed to query past_due tenants", "error", err)
		return
	}
	for _, tenant := range tenants {
		if err := w.admin.AutoSuspend(ctx, strings.TrimSpace(tenant.TenantID)); err != nil {
			w.logger.Error("dunning: auto-suspend failed", "tenant_id", tenant.TenantID, "error", err)
			continue
		}
		w.logger.Info("dunning: tenant auto-suspended after grace period", "tenant_id", tenant.TenantID)
	}
}
