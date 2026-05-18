package timeline

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	domain "github.com/devpablocristo/core/activity/go/timeline/usecases/domain"
)

type Usecases struct {
	repo Repository
	now  func() time.Time
}

func NewUsecases(repo Repository) *Usecases {
	return &Usecases{
		repo: repo,
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

func (u *Usecases) List(ctx context.Context, tenantID, entityType, entityID string, limit int) ([]domain.Entry, error) {
	if strings.TrimSpace(tenantID) == "" {
		return nil, fmt.Errorf("tenant_id is required")
	}
	if strings.TrimSpace(entityType) == "" {
		return nil, fmt.Errorf("entity_type is required")
	}
	if strings.TrimSpace(entityID) == "" {
		return nil, fmt.Errorf("entity_id is required")
	}
	if limit < 0 {
		return nil, fmt.Errorf("limit must be >= 0")
	}
	return u.repo.List(ctx, strings.TrimSpace(tenantID), strings.TrimSpace(entityType), strings.TrimSpace(entityID), limit)
}

func (u *Usecases) Record(ctx context.Context, entry domain.Entry) (domain.Entry, error) {
	if strings.TrimSpace(entry.TenantID) == "" {
		return domain.Entry{}, fmt.Errorf("tenant_id is required")
	}
	if strings.TrimSpace(entry.EntityType) == "" {
		return domain.Entry{}, fmt.Errorf("entity_type is required")
	}
	if strings.TrimSpace(entry.EntityID) == "" {
		return domain.Entry{}, fmt.Errorf("entity_id is required")
	}
	if strings.TrimSpace(entry.EventType) == "" {
		return domain.Entry{}, fmt.Errorf("event_type is required")
	}
	if strings.TrimSpace(entry.Title) == "" {
		return domain.Entry{}, fmt.Errorf("title is required")
	}

	entry.ID = firstNonEmpty(strings.TrimSpace(entry.ID), newID())
	entry.TenantID = strings.TrimSpace(entry.TenantID)
	entry.EntityType = strings.TrimSpace(entry.EntityType)
	entry.EntityID = strings.TrimSpace(entry.EntityID)
	entry.EventType = strings.TrimSpace(entry.EventType)
	entry.Title = strings.TrimSpace(entry.Title)
	entry.Description = strings.TrimSpace(entry.Description)
	entry.Actor = strings.TrimSpace(entry.Actor)
	entry.Metadata = cloneMetadata(entry.Metadata)
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = u.now().UTC()
	} else {
		entry.CreatedAt = entry.CreatedAt.UTC()
	}
	return u.repo.Append(ctx, entry)
}

func cloneMetadata(input map[string]any) map[string]any {
	if len(input) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		out[key] = value
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func newID() string {
	var data [16]byte
	if _, err := rand.Read(data[:]); err != nil {
		return fmt.Sprintf("timeline-%d", time.Now().UTC().UnixNano())
	}
	return hex.EncodeToString(data[:])
}
