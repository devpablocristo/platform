package audit

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	domain "github.com/devpablocristo/core/activity/go/audit/usecases/domain"
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

func (u *Usecases) Append(ctx context.Context, input domain.LogInput) (domain.Entry, error) {
	if strings.TrimSpace(input.TenantID) == "" {
		return domain.Entry{}, fmt.Errorf("tenant_id is required")
	}
	if strings.TrimSpace(input.Action) == "" {
		return domain.Entry{}, fmt.Errorf("action is required")
	}
	if strings.TrimSpace(input.ResourceType) == "" {
		return domain.Entry{}, fmt.Errorf("resource_type is required")
	}

	prevHash, err := u.repo.LastHash(ctx, strings.TrimSpace(input.TenantID))
	if err != nil {
		return domain.Entry{}, fmt.Errorf("lookup last audit hash: %w", err)
	}

	entry := domain.Entry{
		ID:           firstNonEmpty(strings.TrimSpace(input.ID), newID()),
		TenantID:     strings.TrimSpace(input.TenantID),
		Actor:        strings.TrimSpace(input.Actor.Legacy),
		ActorType:    normalizeActorType(input.Actor.Type),
		ActorID:      strings.TrimSpace(input.Actor.ID),
		ActorLabel:   actorLabel(input),
		Action:       strings.TrimSpace(input.Action),
		ResourceType: strings.TrimSpace(input.ResourceType),
		ResourceID:   strings.TrimSpace(input.ResourceID),
		Payload:      clonePayload(input.Payload),
		PrevHash:     prevHash,
		CreatedAt:    normalizeTime(input.CreatedAt, u.now),
	}

	hash, err := buildHash(entry)
	if err != nil {
		return domain.Entry{}, fmt.Errorf("build audit hash: %w", err)
	}
	entry.Hash = hash
	return u.repo.Append(ctx, entry)
}

func (u *Usecases) List(ctx context.Context, tenantID string, limit int) ([]domain.Entry, error) {
	if strings.TrimSpace(tenantID) == "" {
		return nil, fmt.Errorf("tenant_id is required")
	}
	if limit < 0 {
		return nil, fmt.Errorf("limit must be >= 0")
	}
	return u.repo.List(ctx, strings.TrimSpace(tenantID), limit)
}

func (u *Usecases) ExportCSV(ctx context.Context, tenantID string, limit int) (string, error) {
	items, err := u.List(ctx, tenantID, limit)
	if err != nil {
		return "", err
	}
	var out strings.Builder
	writer := csv.NewWriter(&out)
	if err := writer.Write([]string{"id", "tenant_id", "actor", "actor_type", "actor_id", "actor_label", "action", "resource_type", "resource_id", "prev_hash", "hash", "created_at", "payload"}); err != nil {
		return "", fmt.Errorf("write header: %w", err)
	}
	for _, item := range items {
		payload, err := json.Marshal(item.Payload)
		if err != nil {
			return "", fmt.Errorf("marshal payload: %w", err)
		}
		if err := writer.Write([]string{
			item.ID,
			item.TenantID,
			item.Actor,
			item.ActorType,
			item.ActorID,
			item.ActorLabel,
			item.Action,
			item.ResourceType,
			item.ResourceID,
			item.PrevHash,
			item.Hash,
			item.CreatedAt.Format(time.RFC3339),
			string(payload),
		}); err != nil {
			return "", fmt.Errorf("write row: %w", err)
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return "", fmt.Errorf("flush csv: %w", err)
	}
	return out.String(), nil
}

func (u *Usecases) ExportJSONL(ctx context.Context, tenantID string, limit int) (string, error) {
	items, err := u.List(ctx, tenantID, limit)
	if err != nil {
		return "", err
	}
	lines := make([]string, 0, len(items))
	for _, item := range items {
		body, err := json.Marshal(item)
		if err != nil {
			return "", fmt.Errorf("marshal audit entry: %w", err)
		}
		lines = append(lines, string(body))
	}
	return strings.Join(lines, "\n"), nil
}

func actorLabel(input domain.LogInput) string {
	if label := strings.TrimSpace(input.Actor.Label); label != "" {
		return label
	}
	return strings.TrimSpace(input.Actor.Legacy)
}

func normalizeActorType(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "party", "service", "system":
		return strings.ToLower(strings.TrimSpace(raw))
	default:
		return "user"
	}
}

func normalizeTime(value time.Time, now func() time.Time) time.Time {
	if value.IsZero() {
		return now().UTC()
	}
	return value.UTC()
}

func clonePayload(input map[string]any) map[string]any {
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

func buildHash(entry domain.Entry) (string, error) {
	body, err := json.Marshal(struct {
		PrevHash     string         `json:"prev_hash,omitempty"`
		Actor        string         `json:"actor,omitempty"`
		ActorType    string         `json:"actor_type,omitempty"`
		ActorID      string         `json:"actor_id,omitempty"`
		ActorLabel   string         `json:"actor_label,omitempty"`
		Action       string         `json:"action"`
		ResourceType string         `json:"resource_type"`
		ResourceID   string         `json:"resource_id,omitempty"`
		Payload      map[string]any `json:"payload,omitempty"`
	}{
		PrevHash:     entry.PrevHash,
		Actor:        entry.Actor,
		ActorType:    entry.ActorType,
		ActorID:      entry.ActorID,
		ActorLabel:   entry.ActorLabel,
		Action:       entry.Action,
		ResourceType: entry.ResourceType,
		ResourceID:   entry.ResourceID,
		Payload:      entry.Payload,
	})
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:]), nil
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
		return fmt.Sprintf("audit-%d", time.Now().UTC().UnixNano())
	}
	return hex.EncodeToString(data[:])
}
