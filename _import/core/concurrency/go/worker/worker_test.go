package worker_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/devpablocristo/core/concurrency/go/worker"
)

func TestRunPeriodic(t *testing.T) {
	t.Parallel()
	var count atomic.Int32
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		worker.RunPeriodic(ctx, 10*time.Millisecond, "test", func(_ context.Context) {
			count.Add(1)
		})
		close(done)
	}()
	time.Sleep(55 * time.Millisecond)
	cancel()
	<-done
	got := count.Load()
	if got < 3 || got > 7 {
		t.Errorf("expected ~5 executions, got %d", got)
	}
}

func TestRunOnceAndPeriodic(t *testing.T) {
	t.Parallel()
	var count atomic.Int32
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		worker.RunOnceAndPeriodic(ctx, 50*time.Millisecond, "test-once", func(_ context.Context) {
			count.Add(1)
		})
		close(done)
	}()
	// Debe ejecutar inmediatamente (1) + al menos 1 tick
	time.Sleep(80 * time.Millisecond)
	cancel()
	<-done
	got := count.Load()
	if got < 2 {
		t.Errorf("expected >= 2 executions (immediate + ticks), got %d", got)
	}
}

func TestRunPeriodicZeroInterval(t *testing.T) {
	t.Parallel()
	// No debe bloquear con interval <= 0
	done := make(chan struct{})
	go func() {
		worker.RunPeriodic(context.Background(), 0, "zero", func(_ context.Context) {
			t.Error("should not execute")
		})
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Error("RunPeriodic with zero interval should return immediately")
	}
}
