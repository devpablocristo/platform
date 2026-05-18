package timeline

import (
	"context"

	domain "github.com/devpablocristo/core/activity/go/timeline/usecases/domain"
)

type Repository interface {
	List(ctx context.Context, tenantID, entityType, entityID string, limit int) ([]domain.Entry, error)
	Append(ctx context.Context, entry domain.Entry) (domain.Entry, error)
}
