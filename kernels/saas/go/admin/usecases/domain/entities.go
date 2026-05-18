package domain

import "time"

type TenantStatus string

const (
	TenantStatusActive    TenantStatus = "active"
	TenantStatusSuspended TenantStatus = "suspended"
	TenantStatusDeleted   TenantStatus = "deleted"
)

type TenantSettings struct {
	TenantID   string         `json:"tenant_id"`
	PlanCode   string         `json:"plan_code"`
	Status     TenantStatus   `json:"status"`
	DeletedAt  *time.Time     `json:"deleted_at,omitempty"`
	HardLimits map[string]any `json:"hard_limits,omitempty"`
	UpdatedBy  *string        `json:"updated_by,omitempty"`
	UpdatedAt  time.Time      `json:"updated_at"`
	CreatedAt  time.Time      `json:"created_at"`
}

type AdminActivityEvent struct {
	ID           string         `json:"id"`
	TenantID     string         `json:"tenant_id"`
	Actor        *string        `json:"actor,omitempty"`
	Action       string         `json:"action"`
	ResourceType string         `json:"resource_type"`
	ResourceID   *string        `json:"resource_id,omitempty"`
	Payload      map[string]any `json:"payload,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
}

type ProtectedResource struct {
	ID           string    `json:"id"`
	TenantID     string    `json:"tenant_id"`
	Name         string    `json:"name"`
	ResourceType string    `json:"resource_type"`
	MatchValue   string    `json:"match_value"`
	MatchMode    string    `json:"match_mode"`
	Environment  string    `json:"environment"`
	Reason       string    `json:"reason"`
	Enabled      bool      `json:"enabled"`
	CreatedBy    *string   `json:"created_by,omitempty"`
	UpdatedBy    *string   `json:"updated_by,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type RestoreEvidence struct {
	ID             string         `json:"id"`
	TenantID       string         `json:"tenant_id"`
	Environment    string         `json:"environment"`
	System         string         `json:"system"`
	Status         string         `json:"status"`
	SnapshotID     string         `json:"snapshot_id"`
	RestoreTarget  string         `json:"restore_target"`
	StartedAt      *time.Time     `json:"started_at,omitempty"`
	CompletedAt    *time.Time     `json:"completed_at,omitempty"`
	Source         string         `json:"source"`
	ArtifactSHA256 string         `json:"artifact_sha256"`
	Summary        map[string]any `json:"summary,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
}
