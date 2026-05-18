package domain

import "time"

type Entry struct {
	ID          string         `json:"id"`
	TenantID    string         `json:"tenant_id"`
	EntityType  string         `json:"entity_type"`
	EntityID    string         `json:"entity_id"`
	EventType   string         `json:"event_type"`
	Title       string         `json:"title"`
	Description string         `json:"description,omitempty"`
	Actor       string         `json:"actor,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
}
