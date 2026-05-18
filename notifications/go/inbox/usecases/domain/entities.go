package domain

import (
	"encoding/json"
	"time"
)

// Notification representa un aviso persistido en la bandeja in-app de un tenant.
type Notification struct {
	ID          string          `json:"id"`
	TenantID    string          `json:"tenant_id"`
	RecipientID string          `json:"recipient_id"`
	Title       string          `json:"title"`
	Body        string          `json:"body"`
	Kind        string          `json:"kind"`
	EntityType  string          `json:"entity_type,omitempty"`
	EntityID    string          `json:"entity_id,omitempty"`
	Metadata    json.RawMessage `json:"metadata,omitempty"`
	ReadAt      *time.Time      `json:"read_at,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
}
