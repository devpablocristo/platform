package usagemetering

import (
	"context"
	"testing"
	"time"

	usage "github.com/devpablocristo/core/saas/go/usagemetering/usecases/domain"
)

func TestCurrentPeriod(t *testing.T) {
	t.Parallel()

	value := CurrentPeriod(time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC))
	if value != "2026-03" {
		t.Fatalf("unexpected period: %q", value)
	}
}

func TestIncrement(t *testing.T) {
	t.Parallel()

	port := &meteringPort{}
	usecases := NewUseCases(port)
	if err := usecases.Increment(context.Background(), "acme", usage.CounterAPICalls); err != nil {
		t.Fatalf("Increment returned error: %v", err)
	}
	if port.counter != "api_calls" || port.tenantID != "acme" {
		t.Fatalf("unexpected metering input: %#v", port)
	}
}

type meteringPort struct {
	tenantID string
	counter  string
}

func (m *meteringPort) Increment(_ context.Context, tenantID string, counter string) error {
	m.tenantID = tenantID
	m.counter = counter
	return nil
}
