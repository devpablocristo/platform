package webhook

import (
	"context"

	domain "github.com/devpablocristo/core/webhook/go/usecases/domain"
)

type Repository interface {
	GetEndpoint(ctx context.Context, tenantID, endpointID string) (domain.Endpoint, error)
	CreateEndpoint(ctx context.Context, endpoint domain.Endpoint) (domain.Endpoint, error)
	UpdateEndpoint(ctx context.Context, endpoint domain.Endpoint) (domain.Endpoint, error)
	ListEndpointsForEvent(ctx context.Context, tenantID, eventType string) ([]domain.Endpoint, error)
	CreateDelivery(ctx context.Context, delivery domain.Delivery) (domain.Delivery, error)
	UpdateDelivery(ctx context.Context, delivery domain.Delivery) (domain.Delivery, error)
}
