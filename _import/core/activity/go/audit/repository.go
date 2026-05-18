package audit

import (
	"context"

	domain "github.com/devpablocristo/core/activity/go/audit/usecases/domain"
)

type Repository interface {
	LastHash(ctx context.Context, tenantID string) (string, error)
	Append(ctx context.Context, entry domain.Entry) (domain.Entry, error)
	List(ctx context.Context, tenantID string, limit int) ([]domain.Entry, error)
}
