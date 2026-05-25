package lifecycle

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Service is the default ArchiveService implementation. It composes a
// RepositoryPort (mutates the resource's table), an AuditPort (records the
// action), and a PolicyRegistry (enforces per-ResourceType rules).
//
// A single Service instance is *not* tied to a single ResourceType — the
// caller passes ResourceType in each request, and Service dispatches to the
// registered policy + the RepositoryPort registered for that type.
//
// Multi-table dispatch: consumers typically register a separate
// RepositoryPort per ResourceType (e.g. one for customers, one for quotes).
// See NewServiceWithRepos for that pattern. The single-repo constructor
// (NewService) is convenient when all resources share one table.
type Service struct {
	repos    map[string]RepositoryPort
	audit    AuditPort
	policies PolicyRegistry
	clock    Clock
	newID    func() uuid.UUID
}

// NewServiceWithRepos wires a Service against one RepositoryPort per
// ResourceType. Required: at least one entry in repos, plus audit and
// policies. Optional: clock (defaults to time.Now().UTC) and newID
// (defaults to uuid.New).
func NewServiceWithRepos(
	repos map[string]RepositoryPort,
	audit AuditPort,
	policies PolicyRegistry,
	opts ...ServiceOption,
) (*Service, error) {
	if len(repos) == 0 {
		return nil, fmt.Errorf("lifecycle: empty repos map")
	}
	if audit == nil {
		return nil, fmt.Errorf("lifecycle: nil AuditPort")
	}
	if policies == nil {
		return nil, fmt.Errorf("lifecycle: nil PolicyRegistry")
	}
	s := &Service{
		repos:    repos,
		audit:    audit,
		policies: policies,
		clock:    func() time.Time { return time.Now().UTC() },
		newID:    uuid.New,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s, nil
}

// ServiceOption customizes the Service at construction time.
type ServiceOption func(*Service)

// WithClock overrides the time source (test injection).
func WithClock(c Clock) ServiceOption {
	return func(s *Service) {
		if c != nil {
			s.clock = c
		}
	}
}

// WithIDGenerator overrides the audit ID generator (test injection).
func WithIDGenerator(gen func() uuid.UUID) ServiceOption {
	return func(s *Service) {
		if gen != nil {
			s.newID = gen
		}
	}
}

func (s *Service) repoFor(resourceType string) (RepositoryPort, error) {
	r, ok := s.repos[resourceType]
	if !ok {
		return nil, fmt.Errorf("lifecycle: no RepositoryPort registered for ResourceType %q", resourceType)
	}
	return r, nil
}

// SoftDelete archives a resource. Steps:
//  1. Resolve policy for req.ResourceType.
//  2. Check AllowArchive and RequireReason.
//  3. Run policy.ValidateRelations if defined.
//  4. Resolve RepositoryPort and call SoftDelete (idempotency: a 2nd call
//     against an already-archived resource returns domainerr.NotFound — see
//     SoftDeleter.SoftDelete).
//  5. Append audit record.
func (s *Service) SoftDelete(ctx context.Context, req *ArchiveRequest) error {
	if req == nil {
		return fmt.Errorf("lifecycle: nil ArchiveRequest")
	}
	policy, err := s.policies.Get(req.ResourceType)
	if err != nil {
		return err
	}
	if !policy.AllowArchive {
		return ErrArchiveNotAllowed
	}
	if policy.RequireReason && req.Reason == "" {
		return ErrReasonRequired
	}
	if policy.ValidateRelations != nil {
		if err := policy.ValidateRelations(ctx, req.TenantID, req.ResourceID); err != nil {
			return err
		}
	}
	repo, err := s.repoFor(req.ResourceType)
	if err != nil {
		return err
	}
	now := s.clock()
	if err := repo.SoftDelete(ctx, req.TenantID, req.ResourceID, now); err != nil {
		return err
	}
	return s.appendAudit(ctx, req, policy, ActionArchive, now)
}

// Restore brings an archived resource back. Returns domainerr.NotFound if
// it wasn't archived.
func (s *Service) Restore(ctx context.Context, req *RestoreRequest) error {
	if req == nil {
		return fmt.Errorf("lifecycle: nil RestoreRequest")
	}
	policy, err := s.policies.Get(req.ResourceType)
	if err != nil {
		return err
	}
	// AllowArchive=false also disables Restore (you can't restore what you
	// can't archive). Consumers wanting one-way archival should set AllowArchive=true
	// but reject Restore via ValidateRelations on the consumer side.
	if !policy.AllowArchive {
		return ErrArchiveNotAllowed
	}
	repo, err := s.repoFor(req.ResourceType)
	if err != nil {
		return err
	}
	if err := repo.Restore(ctx, req.TenantID, req.ResourceID); err != nil {
		return err
	}
	now := s.clock()
	reason := optionalReason(req.Reason)
	return s.audit.Append(ctx, ArchiveAudit{
		ID:           s.newID(),
		TenantID:     req.TenantID,
		ResourceType: req.ResourceType,
		ResourceID:   req.ResourceID,
		Action:       ActionRestore,
		OccurredAt:   now,
		Actor:        req.Actor,
		Reason:       reason,
	})
}

// HardDelete permanently removes the resource. Enforces policy.AllowHardDelete.
// When req.MustBeArchived = true, the resource must be already archived (per
// policy intent of soft-then-hard).
func (s *Service) HardDelete(ctx context.Context, req *HardDeleteRequest) error {
	if req == nil {
		return fmt.Errorf("lifecycle: nil HardDeleteRequest")
	}
	policy, err := s.policies.Get(req.ResourceType)
	if err != nil {
		return err
	}
	if !policy.AllowHardDelete {
		return ErrHardDeleteNotAllowed
	}
	if policy.RequireReason && req.Reason == "" {
		return ErrReasonRequired
	}
	repo, err := s.repoFor(req.ResourceType)
	if err != nil {
		return err
	}
	if req.MustBeArchived {
		archived, err := repo.IsArchived(ctx, req.TenantID, req.ResourceID)
		if err != nil {
			return err
		}
		if !archived {
			return ErrArchiveNotAllowed // misuse: caller wanted soft-first
		}
	}
	if err := repo.HardDelete(ctx, req.TenantID, req.ResourceID); err != nil {
		return err
	}
	now := s.clock()
	reason := optionalReason(req.Reason)
	return s.audit.Append(ctx, ArchiveAudit{
		ID:           s.newID(),
		TenantID:     req.TenantID,
		ResourceType: req.ResourceType,
		ResourceID:   req.ResourceID,
		Action:       ActionHardDelete,
		OccurredAt:   now,
		Actor:        req.Actor,
		Reason:       reason,
	})
}

// BulkArchive archives multiple resources of the same ResourceType. Each ID
// is processed independently; per-ID errors are accumulated and returned in
// the result slice. A non-nil top-level error indicates a configuration
// problem (e.g. unknown ResourceType, policy lookup failed).
func (s *Service) BulkArchive(
	ctx context.Context,
	resourceType string,
	tenantID string,
	actor, reason string,
	ids []uuid.UUID,
) ([]BulkArchiveResult, error) {
	policy, err := s.policies.Get(resourceType)
	if err != nil {
		return nil, err
	}
	if !policy.AllowArchive {
		return nil, ErrArchiveNotAllowed
	}
	if policy.RequireReason && reason == "" {
		return nil, ErrReasonRequired
	}
	batchID := s.newID()
	results := make([]BulkArchiveResult, 0, len(ids))
	for _, id := range ids {
		req := &ArchiveRequest{
			ResourceType: resourceType,
			ResourceID:   id,
			TenantID:     tenantID,
			Actor:        actor,
			Reason:       reason,
			BatchID:      &batchID,
		}
		results = append(results, BulkArchiveResult{
			ResourceID: id,
			Err:        s.SoftDelete(ctx, req),
		})
	}
	return results, nil
}

func (s *Service) appendAudit(
	ctx context.Context,
	req *ArchiveRequest,
	policy *ArchivePolicy,
	action Action,
	occurredAt time.Time,
) error {
	var retention *time.Time
	if policy.RetentionDays > 0 {
		t := occurredAt.AddDate(0, 0, policy.RetentionDays)
		retention = &t
	}
	reason := optionalReason(req.Reason)
	return s.audit.Append(ctx, ArchiveAudit{
		ID:               s.newID(),
		TenantID:         req.TenantID,
		ResourceType:     req.ResourceType,
		ResourceID:       req.ResourceID,
		Action:           action,
		OccurredAt:       occurredAt,
		Actor:            req.Actor,
		Reason:           reason,
		BatchID:          req.BatchID,
		RetentionExpires: retention,
	})
}

func optionalReason(reason string) *string {
	if reason == "" {
		return nil
	}
	return &reason
}
