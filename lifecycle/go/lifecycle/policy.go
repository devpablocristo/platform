package lifecycle

import (
	"context"

	"github.com/devpablocristo/platform/errors/go/domainerr"
	"github.com/google/uuid"
)

// ArchivePolicy describes the rules applied to a single ResourceType.
//
// Instances of ArchivePolicy live in the consumer (e.g. pymes) — this struct
// is the *mechanism*; the values are decisions of the product. See § Invariantes
// I3 in the migration plan.
type ArchivePolicy struct {
	ResourceType    string
	AllowArchive    bool
	AllowHardDelete bool
	// RequireReason: when true, ArchiveRequest.Reason must be non-empty.
	RequireReason bool
	// ValidateRelations runs before SoftDelete; if it returns a non-nil error,
	// the archive is rejected. Caller-defined (e.g. "no se puede archivar
	// si tiene relaciones activas").
	ValidateRelations func(ctx context.Context, tenantID, resourceID uuid.UUID) error
	// RetentionDays: 0 = retain forever; >0 = PurgeExpired hard-deletes
	// records whose ArchivedAt + RetentionDays has elapsed.
	RetentionDays int
	// NotifyOnArchive: when true, emit a notification event for the audit
	// integration to broadcast. The dispatch itself is consumer-controlled
	// (we don't know which transport).
	NotifyOnArchive bool
}

// PolicyRegistry resolves ArchivePolicy by ResourceType. Default
// implementation (NewStaticPolicyRegistry) is a plain map; consumers can
// also implement a registry that reads from configuration or a DB.
type PolicyRegistry interface {
	Get(resourceType string) (*ArchivePolicy, error)
}

// ErrPolicyNotFound is returned by PolicyRegistry.Get when the requested
// ResourceType has no registered policy. It maps to domainerr.NotFound.
var ErrPolicyNotFound = domainerr.NotFound("archive_policy")

// ErrArchiveNotAllowed is returned when AllowArchive is false.
var ErrArchiveNotAllowed = domainerr.Conflict("archive_not_allowed")

// ErrHardDeleteNotAllowed is returned when AllowHardDelete is false.
var ErrHardDeleteNotAllowed = domainerr.Conflict("hard_delete_not_allowed")

// ErrReasonRequired is returned when policy requires Reason and none is given.
var ErrReasonRequired = domainerr.Validation("reason_required")

// staticPolicyRegistry is a simple in-memory map keyed by ResourceType.
type staticPolicyRegistry struct {
	policies map[string]*ArchivePolicy
}

// NewStaticPolicyRegistry builds a PolicyRegistry from a list of policies.
// Duplicate ResourceTypes cause a panic at construction time (configuration
// error, not a runtime error).
func NewStaticPolicyRegistry(policies ...*ArchivePolicy) PolicyRegistry {
	r := &staticPolicyRegistry{policies: make(map[string]*ArchivePolicy, len(policies))}
	for _, p := range policies {
		if p == nil {
			continue
		}
		if _, dup := r.policies[p.ResourceType]; dup {
			panic("lifecycle: duplicate ArchivePolicy for ResourceType " + p.ResourceType)
		}
		r.policies[p.ResourceType] = p
	}
	return r
}

func (r *staticPolicyRegistry) Get(resourceType string) (*ArchivePolicy, error) {
	p, ok := r.policies[resourceType]
	if !ok {
		return nil, ErrPolicyNotFound
	}
	return p, nil
}
