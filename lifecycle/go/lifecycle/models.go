// Package lifecycle provides domain-agnostic primitives for the canonical
// resource lifecycle:
//
//	active -> archived -> active
//	active/archived -> trashed -> active
//	trashed -> purged
//
// Archive is not deletion. Archived resources are retained but excluded from
// active workflows by default. Trash is the reversible delete state. Purge is
// irreversible deletion.
package lifecycle

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// LifecycleState is the canonical state of a resource.
type LifecycleState string

const (
	StateActive   LifecycleState = "active"
	StateArchived LifecycleState = "archived"
	StateTrashed  LifecycleState = "trashed"
	StatePurged   LifecycleState = "purged"
)

// ParticipatesInAutomation returns true when resources in state should be
// included by default in automation, derived intelligence and active workflows.
func ParticipatesInAutomation(state LifecycleState) bool {
	return state == "" || state == StateActive
}

// Action is the action recorded in audit. Callers can extend with their own
// values because the field is a plain string.
type Action string

const (
	ActionArchive   Action = "archive"
	ActionUnarchive Action = "unarchive"
	ActionTrash     Action = "trash"
	ActionRestore   Action = "restore"
	ActionPurge     Action = "purge"
)

// ArchiveRequest captures the input to Archive and BulkArchive.
type ArchiveRequest struct {
	ResourceType string
	ResourceID   uuid.UUID
	TenantID     string
	Actor        string
	Reason       string
	BatchID      *uuid.UUID
}

// UnarchiveRequest captures the input to Unarchive.
type UnarchiveRequest struct {
	ResourceType string
	ResourceID   uuid.UUID
	TenantID     string
	Actor        string
	Reason       string
}

// TrashRequest captures the input to Trash and BulkTrash.
type TrashRequest struct {
	ResourceType string
	ResourceID   uuid.UUID
	TenantID     string
	Actor        string
	Reason       string
	BatchID      *uuid.UUID
}

// RestoreRequest captures the input to Restore, which moves a trashed resource
// back to active.
type RestoreRequest struct {
	ResourceType string
	ResourceID   uuid.UUID
	TenantID     string
	Actor        string
	Reason       string
}

// PurgeRequest captures the input to Purge.
//
// MustBeTrashed should be true for user-facing products. Set it to false only
// for explicit administrative or compliance workflows that can purge directly.
type PurgeRequest struct {
	ResourceType  string
	ResourceID    uuid.UUID
	TenantID      string
	Actor         string
	Reason        string
	MustBeTrashed bool
}

// LifecycleAudit is the append-only audit record produced by every action.
type LifecycleAudit struct {
	ID               uuid.UUID
	TenantID         string
	ResourceType     string
	ResourceID       uuid.UUID
	Action           Action
	OccurredAt       time.Time
	Actor            string
	Reason           *string
	BatchID          *uuid.UUID
	FromState        LifecycleState
	ToState          LifecycleState
	RetentionExpires *time.Time
}

// LifecycleQuery filters lifecycle listings.
type LifecycleQuery struct {
	TenantID     string
	ResourceType string // optional; empty = all
	State        LifecycleState
	Since        *time.Time
	Until        *time.Time
	Actor        string // optional
	BatchID      *uuid.UUID
	Limit        int // 0 = caller default
}

// BulkArchiveResult reports per-ID outcome of BulkArchive.
type BulkArchiveResult struct {
	ResourceID uuid.UUID
	Err        error // nil on success
}

// BulkTrashResult reports per-ID outcome of BulkTrash.
type BulkTrashResult struct {
	ResourceID uuid.UUID
	Err        error // nil on success
}

// AuditPort persists lifecycle audit records.
type AuditPort interface {
	Append(ctx context.Context, entry LifecycleAudit) error
}

// RepositoryPort persists lifecycle transitions on the resource's own table.
//
// Implementations must scope every operation to TenantID for multi-tenant
// safety.
type RepositoryPort interface {
	Archive(ctx context.Context, tenantID string, resourceID uuid.UUID, at time.Time) error
	Unarchive(ctx context.Context, tenantID string, resourceID uuid.UUID) error
	Trash(ctx context.Context, tenantID string, resourceID uuid.UUID, at time.Time, purgeAfter *time.Time) error
	Restore(ctx context.Context, tenantID string, resourceID uuid.UUID) error
	Purge(ctx context.Context, tenantID string, resourceID uuid.UUID) error
	State(ctx context.Context, tenantID string, resourceID uuid.UUID) (LifecycleState, error)
}

// Clock is used to inject time for tests. Default: time.Now().UTC.
type Clock func() time.Time
