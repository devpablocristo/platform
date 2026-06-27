package domain

import "time"

type OrgStatus string

const (
	OrgStatusActive    OrgStatus = "active"
	OrgStatusSuspended OrgStatus = "suspended"
	OrgStatusDeleted   OrgStatus = "deleted"
)

type OrgSettings struct {
	OrgID      string         `json:"org_id"`
	TenantID   string         `json:"-"`
	PlanCode   string         `json:"plan_code"`
	Status     OrgStatus      `json:"status"`
	DeletedAt  *time.Time     `json:"deleted_at,omitempty"`
	HardLimits map[string]any `json:"hard_limits,omitempty"`
	UpdatedBy  *string        `json:"updated_by,omitempty"`
	UpdatedAt  time.Time      `json:"updated_at"`
	CreatedAt  time.Time      `json:"created_at"`
}

func (s OrgSettings) EffectiveOrgID() string {
	if s.OrgID != "" {
		return s.OrgID
	}
	return s.TenantID
}

type AdminActivityEvent struct {
	ID           string         `json:"id"`
	OrgID        string         `json:"org_id"`
	TenantID     string         `json:"-"`
	Actor        *string        `json:"actor,omitempty"`
	Action       string         `json:"action"`
	ResourceType string         `json:"resource_type"`
	ResourceID   *string        `json:"resource_id,omitempty"`
	Payload      map[string]any `json:"payload,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
}

type ProtectedResource struct {
	ID           string    `json:"id"`
	OrgID        string    `json:"org_id"`
	TenantID     string    `json:"-"`
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
	OrgID          string         `json:"org_id"`
	TenantID       string         `json:"-"`
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

// Legacy aliases kept while existing consumers migrate.
type TenantStatus = OrgStatus

const (
	TenantStatusActive    = OrgStatusActive
	TenantStatusSuspended = OrgStatusSuspended
	TenantStatusDeleted   = OrgStatusDeleted
)

type TenantSettings = OrgSettings
