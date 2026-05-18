package candidates

import (
	"context"
	"fmt"
	"strings"
	"time"

	domain "github.com/devpablocristo/core/notifications/go/candidates/usecases/domain"
)

type Usecases struct {
	recorder Recorder
	notifier Notifier
	reader   Reader
	now      func() time.Time
}

func NewUsecases(repo Repository) *Usecases {
	return &Usecases{
		recorder: repo,
		notifier: repo,
		reader:   repo,
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

func NewWriteUsecases(recorder Recorder, notifier Notifier) *Usecases {
	return &Usecases{
		recorder: recorder,
		notifier: notifier,
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

func NewReadUsecases(reader Reader) *Usecases {
	return &Usecases{
		reader: reader,
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

func (u *Usecases) Record(ctx context.Context, input domain.UpsertInput) (domain.Candidate, bool, error) {
	input.OrgID = strings.TrimSpace(input.OrgID)
	if input.OrgID == "" {
		return domain.Candidate{}, false, fmt.Errorf("org_id is required")
	}
	input.Kind = firstNonEmpty(input.Kind, "insight")
	input.EventType = strings.TrimSpace(input.EventType)
	if input.EventType == "" {
		return domain.Candidate{}, false, fmt.Errorf("event_type is required")
	}
	input.EntityType = strings.TrimSpace(input.EntityType)
	if input.EntityType == "" {
		return domain.Candidate{}, false, fmt.Errorf("entity_type is required")
	}
	input.EntityID = strings.TrimSpace(input.EntityID)
	if input.EntityID == "" {
		return domain.Candidate{}, false, fmt.Errorf("entity_id is required")
	}
	input.Fingerprint = strings.TrimSpace(input.Fingerprint)
	if input.Fingerprint == "" {
		return domain.Candidate{}, false, fmt.Errorf("fingerprint is required")
	}
	input.Severity = firstNonEmpty(input.Severity, "info")
	input.Title = strings.TrimSpace(input.Title)
	if input.Title == "" {
		return domain.Candidate{}, false, fmt.Errorf("title is required")
	}
	input.Body = strings.TrimSpace(input.Body)
	if input.Body == "" {
		return domain.Candidate{}, false, fmt.Errorf("body is required")
	}
	input.Actor = strings.TrimSpace(input.Actor)
	if input.Now.IsZero() {
		input.Now = u.now().UTC()
	} else {
		input.Now = input.Now.UTC()
	}
	if len(input.Evidence) == 0 {
		input.Evidence = map[string]any{}
	}
	if u.recorder == nil {
		return domain.Candidate{}, false, fmt.Errorf("recorder is required")
	}
	return u.recorder.Upsert(ctx, input)
}

func (u *Usecases) MarkNotified(ctx context.Context, orgID, candidateID string) error {
	orgID = strings.TrimSpace(orgID)
	if orgID == "" {
		return fmt.Errorf("org_id is required")
	}
	candidateID = strings.TrimSpace(candidateID)
	if candidateID == "" {
		return fmt.Errorf("candidate_id is required")
	}
	if u.notifier == nil {
		return fmt.Errorf("notifier is required")
	}
	return u.notifier.MarkNotified(ctx, orgID, candidateID, u.now().UTC())
}

func (u *Usecases) List(ctx context.Context, orgID string, limit int) ([]domain.Candidate, error) {
	orgID = strings.TrimSpace(orgID)
	if orgID == "" {
		return nil, fmt.Errorf("org_id is required")
	}
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	if u.reader == nil {
		return nil, fmt.Errorf("reader is required")
	}
	return u.reader.ListByTenant(ctx, orgID, limit)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
