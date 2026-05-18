package candidates

import (
	"context"
	"time"

	domain "github.com/devpablocristo/core/notifications/go/candidates/usecases/domain"
)

type Repository interface {
	Recorder
	Notifier
	Reader
}

type Recorder interface {
	Upsert(ctx context.Context, in domain.UpsertInput) (domain.Candidate, bool, error)
}

type Notifier interface {
	MarkNotified(ctx context.Context, orgID, candidateID string, notifiedAt time.Time) error
}

type Reader interface {
	ListByTenant(ctx context.Context, orgID string, limit int) ([]domain.Candidate, error)
}
