package lifecycle

import (
	"context"

	"github.com/devpablocristo/platform/errors/go/domainerr"
	"github.com/google/uuid"
)

// LifecyclePolicy describes the rules applied to a single ResourceType.
//
// Instances live in the consumer product. This package provides mechanism; the
// product decides which resources can be archived, trashed or purged.
type LifecyclePolicy struct {
	ResourceType string

	AllowArchive bool
	AllowTrash   bool
	AllowPurge   bool

	// RequireReason applies to archive, trash and purge. Consumers can enforce
	// narrower rules by wrapping Service or using validators.
	RequireReason bool

	ValidateArchive func(ctx context.Context, tenantID string, resourceID uuid.UUID) error
	ValidateTrash   func(ctx context.Context, tenantID string, resourceID uuid.UUID) error
	ValidatePurge   func(ctx context.Context, tenantID string, resourceID uuid.UUID) error

	// RetentionDays: 0 = retain forever in trash; >0 = Trash stores
	// purge_after = now + RetentionDays.
	RetentionDays int

	// NotifyOnArchive is metadata for consumers that dispatch notifications from
	// audit records. The dispatch itself is consumer-controlled.
	NotifyOnArchive bool
}

// PolicyRegistry resolves LifecyclePolicy by ResourceType.
type PolicyRegistry interface {
	Get(resourceType string) (*LifecyclePolicy, error)
}

var ErrPolicyNotFound = domainerr.NotFound("lifecycle_policy")
var ErrArchiveNotAllowed = domainerr.Conflict("archive_not_allowed")
var ErrTrashNotAllowed = domainerr.Conflict("trash_not_allowed")
var ErrPurgeNotAllowed = domainerr.Conflict("purge_not_allowed")
var ErrMustBeTrashed = domainerr.Conflict("must_be_trashed")
var ErrReasonRequired = domainerr.Validation("reason_required")

type staticPolicyRegistry struct {
	policies map[string]*LifecyclePolicy
}

// NewStaticPolicyRegistry builds a PolicyRegistry from a list of policies.
// Duplicate ResourceTypes cause a panic at construction time.
func NewStaticPolicyRegistry(policies ...*LifecyclePolicy) PolicyRegistry {
	r := &staticPolicyRegistry{policies: make(map[string]*LifecyclePolicy, len(policies))}
	for _, p := range policies {
		if p == nil {
			continue
		}
		if _, dup := r.policies[p.ResourceType]; dup {
			panic("lifecycle: duplicate LifecyclePolicy for ResourceType " + p.ResourceType)
		}
		r.policies[p.ResourceType] = p
	}
	return r
}

func (r *staticPolicyRegistry) Get(resourceType string) (*LifecyclePolicy, error) {
	p, ok := r.policies[resourceType]
	if !ok {
		return nil, ErrPolicyNotFound
	}
	return p, nil
}
