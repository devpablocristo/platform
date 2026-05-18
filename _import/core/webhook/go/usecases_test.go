package webhook

import (
	"context"
	"strings"
	"testing"
	"time"

	domain "github.com/devpablocristo/core/webhook/go/usecases/domain"
)

func TestCreateEndpointNormalizesFields(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{}
	usecases := NewUsecases(repo)
	usecases.now = func() time.Time { return time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC) }

	endpoint, err := usecases.CreateEndpoint(context.Background(), domain.Endpoint{
		TenantID: "acme",
		URL:      "https://hooks.example.com/events",
		Events:   []string{"invoice.created", "invoice.created", " invoice.paid "},
	})
	if err != nil {
		t.Fatalf("CreateEndpoint returned error: %v", err)
	}
	if endpoint.ID == "" {
		t.Fatal("expected generated id")
	}
	if endpoint.Secret == "" {
		t.Fatal("expected generated secret")
	}
	if len(endpoint.Events) != 2 {
		t.Fatalf("unexpected events: %#v", endpoint.Events)
	}
}

func TestBuildDispatchIncludesSignature(t *testing.T) {
	t.Parallel()

	usecases := NewUsecases(&fakeRepository{})
	usecases.now = func() time.Time { return time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC) }

	request, delivery, err := usecases.BuildDispatch(context.Background(), domain.Endpoint{
		ID:       "ep_1",
		URL:      "https://hooks.example.com/events",
		Secret:   "secret",
		IsActive: true,
	}, "invoice.created", map[string]any{"amount": 10}, 2)
	if err != nil {
		t.Fatalf("BuildDispatch returned error: %v", err)
	}
	if request.Method != "POST" {
		t.Fatalf("unexpected method: %q", request.Method)
	}
	if !strings.HasPrefix(request.Headers["X-Webhook-Signature"], "sha256=") {
		t.Fatalf("unexpected signature: %q", request.Headers["X-Webhook-Signature"])
	}
	if delivery.Attempts != 2 {
		t.Fatalf("unexpected attempts: %d", delivery.Attempts)
	}
}

func TestNextRetryStopsAtMaxAttempts(t *testing.T) {
	t.Parallel()

	usecases := NewUsecases(&fakeRepository{})
	next, ok := usecases.NextRetry(time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC), 5)
	if ok {
		t.Fatalf("expected no retry, got %s", next)
	}
}

type fakeRepository struct {
	endpoint domain.Endpoint
	delivery domain.Delivery
}

func (r *fakeRepository) GetEndpoint(_ context.Context, _, _ string) (domain.Endpoint, error) {
	return r.endpoint, nil
}

func (r *fakeRepository) CreateEndpoint(_ context.Context, endpoint domain.Endpoint) (domain.Endpoint, error) {
	r.endpoint = endpoint
	return endpoint, nil
}

func (r *fakeRepository) UpdateEndpoint(_ context.Context, endpoint domain.Endpoint) (domain.Endpoint, error) {
	r.endpoint = endpoint
	return endpoint, nil
}

func (r *fakeRepository) ListEndpointsForEvent(_ context.Context, _, _ string) ([]domain.Endpoint, error) {
	return []domain.Endpoint{r.endpoint}, nil
}

func (r *fakeRepository) CreateDelivery(_ context.Context, delivery domain.Delivery) (domain.Delivery, error) {
	r.delivery = delivery
	return delivery, nil
}

func (r *fakeRepository) UpdateDelivery(_ context.Context, delivery domain.Delivery) (domain.Delivery, error) {
	r.delivery = delivery
	return delivery, nil
}
