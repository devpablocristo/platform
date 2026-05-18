// Package worker provides a reusable periodic background task runner.
package worker

import (
	"context"
	"log/slog"
	"time"
)

// RunPeriodic ejecuta fn cada interval hasta que ctx se cancele.
// name se usa en los logs para identificar el worker.
func RunPeriodic(ctx context.Context, interval time.Duration, name string, fn func(context.Context)) {
	if interval <= 0 {
		slog.Warn("worker not started: interval <= 0", "worker", name)
		return
	}
	slog.Info("worker started", "worker", name, "interval", interval)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			slog.Info("worker stopped", "worker", name)
			return
		case <-ticker.C:
			fn(ctx)
		}
	}
}

// RunOnceAndPeriodic ejecuta fn inmediatamente y luego cada interval.
func RunOnceAndPeriodic(ctx context.Context, interval time.Duration, name string, fn func(context.Context)) {
	if interval <= 0 {
		slog.Warn("worker not started: interval <= 0", "worker", name)
		return
	}
	slog.Info("worker started", "worker", name, "interval", interval)
	fn(ctx)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			slog.Info("worker stopped", "worker", name)
			return
		case <-ticker.C:
			fn(ctx)
		}
	}
}
