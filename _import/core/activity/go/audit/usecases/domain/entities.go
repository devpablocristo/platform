package domain

import (
	"time"

	kernel "github.com/devpablocristo/core/activity/go/kernel/usecases/domain"
)

type Entry struct {
	ID           string         `json:"id"`
	TenantID     string         `json:"tenant_id"`
	Actor        string         `json:"actor,omitempty"`
	ActorType    string         `json:"actor_type,omitempty"`
	ActorID      string         `json:"actor_id,omitempty"`
	ActorLabel   string         `json:"actor_label,omitempty"`
	Action       string         `json:"action"`
	ResourceType string         `json:"resource_type"`
	ResourceID   string         `json:"resource_id,omitempty"`
	Payload      map[string]any `json:"payload,omitempty"`
	PrevHash     string         `json:"prev_hash,omitempty"`
	Hash         string         `json:"hash"`
	CreatedAt    time.Time      `json:"created_at"`
}

type LogInput struct {
	ID           string
	TenantID     string
	Actor        kernel.ActorRef
	Action       string
	ResourceType string
	ResourceID   string
	Payload      map[string]any
	CreatedAt    time.Time
}
