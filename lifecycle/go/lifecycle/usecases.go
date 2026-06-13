package lifecycle

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Service composes repositories that mutate resource tables, an AuditPort and
// a PolicyRegistry. A single Service can handle multiple ResourceTypes.
type Service struct {
	repos    map[string]RepositoryPort
	audit    AuditPort
	policies PolicyRegistry
	clock    Clock
	newID    func() uuid.UUID
}

// NewServiceWithRepos wires a Service against one RepositoryPort per
// ResourceType.
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

// WithClock overrides the time source.
func WithClock(c Clock) ServiceOption {
	return func(s *Service) {
		if c != nil {
			s.clock = c
		}
	}
}

// WithIDGenerator overrides the audit ID generator.
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

// Archive moves an active resource to the archived state.
func (s *Service) Archive(ctx context.Context, req *ArchiveRequest) error {
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
	if err := requireReason(policy, req.Reason); err != nil {
		return err
	}
	if policy.ValidateArchive != nil {
		if err := policy.ValidateArchive(ctx, req.TenantID, req.ResourceID); err != nil {
			return err
		}
	}
	repo, err := s.repoFor(req.ResourceType)
	if err != nil {
		return err
	}
	now := s.clock()
	if err := repo.Archive(ctx, req.TenantID, req.ResourceID, now); err != nil {
		return err
	}
	return s.appendAudit(ctx, auditInput{
		tenantID: req.TenantID, resourceType: req.ResourceType, resourceID: req.ResourceID,
		action: ActionArchive, occurredAt: now, actor: req.Actor, reason: req.Reason,
		batchID: req.BatchID, fromState: StateActive, toState: StateArchived,
	})
}

// Unarchive moves an archived resource back to active.
func (s *Service) Unarchive(ctx context.Context, req *UnarchiveRequest) error {
	if req == nil {
		return fmt.Errorf("lifecycle: nil UnarchiveRequest")
	}
	policy, err := s.policies.Get(req.ResourceType)
	if err != nil {
		return err
	}
	if !policy.AllowArchive {
		return ErrArchiveNotAllowed
	}
	repo, err := s.repoFor(req.ResourceType)
	if err != nil {
		return err
	}
	if err := repo.Unarchive(ctx, req.TenantID, req.ResourceID); err != nil {
		return err
	}
	now := s.clock()
	return s.appendAudit(ctx, auditInput{
		tenantID: req.TenantID, resourceType: req.ResourceType, resourceID: req.ResourceID,
		action: ActionUnarchive, occurredAt: now, actor: req.Actor, reason: req.Reason,
		fromState: StateArchived, toState: StateActive,
	})
}

// Trash moves a resource to the reversible delete state.
func (s *Service) Trash(ctx context.Context, req *TrashRequest) error {
	if req == nil {
		return fmt.Errorf("lifecycle: nil TrashRequest")
	}
	policy, err := s.policies.Get(req.ResourceType)
	if err != nil {
		return err
	}
	if !policy.AllowTrash {
		return ErrTrashNotAllowed
	}
	if err := requireReason(policy, req.Reason); err != nil {
		return err
	}
	if policy.ValidateTrash != nil {
		if err := policy.ValidateTrash(ctx, req.TenantID, req.ResourceID); err != nil {
			return err
		}
	}
	repo, err := s.repoFor(req.ResourceType)
	if err != nil {
		return err
	}
	now := s.clock()
	var purgeAfter *time.Time
	if policy.RetentionDays > 0 {
		t := now.AddDate(0, 0, policy.RetentionDays)
		purgeAfter = &t
	}
	if err := repo.Trash(ctx, req.TenantID, req.ResourceID, now, purgeAfter); err != nil {
		return err
	}
	return s.appendAudit(ctx, auditInput{
		tenantID: req.TenantID, resourceType: req.ResourceType, resourceID: req.ResourceID,
		action: ActionTrash, occurredAt: now, actor: req.Actor, reason: req.Reason,
		batchID: req.BatchID, fromState: StateActive, toState: StateTrashed,
		retentionExpires: purgeAfter,
	})
}

// Restore moves a trashed resource back to active.
func (s *Service) Restore(ctx context.Context, req *RestoreRequest) error {
	if req == nil {
		return fmt.Errorf("lifecycle: nil RestoreRequest")
	}
	policy, err := s.policies.Get(req.ResourceType)
	if err != nil {
		return err
	}
	if !policy.AllowTrash {
		return ErrTrashNotAllowed
	}
	repo, err := s.repoFor(req.ResourceType)
	if err != nil {
		return err
	}
	if err := repo.Restore(ctx, req.TenantID, req.ResourceID); err != nil {
		return err
	}
	now := s.clock()
	return s.appendAudit(ctx, auditInput{
		tenantID: req.TenantID, resourceType: req.ResourceType, resourceID: req.ResourceID,
		action: ActionRestore, occurredAt: now, actor: req.Actor, reason: req.Reason,
		fromState: StateTrashed, toState: StateActive,
	})
}

// Purge permanently removes the resource. Prefer MustBeTrashed=true for
// user-facing products.
func (s *Service) Purge(ctx context.Context, req *PurgeRequest) error {
	if req == nil {
		return fmt.Errorf("lifecycle: nil PurgeRequest")
	}
	policy, err := s.policies.Get(req.ResourceType)
	if err != nil {
		return err
	}
	if !policy.AllowPurge {
		return ErrPurgeNotAllowed
	}
	if err := requireReason(policy, req.Reason); err != nil {
		return err
	}
	if policy.ValidatePurge != nil {
		if err := policy.ValidatePurge(ctx, req.TenantID, req.ResourceID); err != nil {
			return err
		}
	}
	repo, err := s.repoFor(req.ResourceType)
	if err != nil {
		return err
	}
	fromState, err := repo.State(ctx, req.TenantID, req.ResourceID)
	if err != nil {
		return err
	}
	if req.MustBeTrashed && fromState != StateTrashed {
		return ErrMustBeTrashed
	}
	if err := repo.Purge(ctx, req.TenantID, req.ResourceID); err != nil {
		return err
	}
	now := s.clock()
	return s.appendAudit(ctx, auditInput{
		tenantID: req.TenantID, resourceType: req.ResourceType, resourceID: req.ResourceID,
		action: ActionPurge, occurredAt: now, actor: req.Actor, reason: req.Reason,
		fromState: fromState, toState: StatePurged,
	})
}

// BulkArchive archives multiple resources of the same ResourceType. Each ID is
// processed independently; per-ID errors are accumulated and returned.
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
	if err := requireReason(policy, reason); err != nil {
		return nil, err
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
			Err:        s.Archive(ctx, req),
		})
	}
	return results, nil
}

// BulkTrash trashes multiple resources of the same ResourceType. Each ID is
// processed independently; per-ID errors are accumulated and returned.
func (s *Service) BulkTrash(
	ctx context.Context,
	resourceType string,
	tenantID string,
	actor, reason string,
	ids []uuid.UUID,
) ([]BulkTrashResult, error) {
	policy, err := s.policies.Get(resourceType)
	if err != nil {
		return nil, err
	}
	if !policy.AllowTrash {
		return nil, ErrTrashNotAllowed
	}
	if err := requireReason(policy, reason); err != nil {
		return nil, err
	}
	batchID := s.newID()
	results := make([]BulkTrashResult, 0, len(ids))
	for _, id := range ids {
		req := &TrashRequest{
			ResourceType: resourceType,
			ResourceID:   id,
			TenantID:     tenantID,
			Actor:        actor,
			Reason:       reason,
			BatchID:      &batchID,
		}
		results = append(results, BulkTrashResult{
			ResourceID: id,
			Err:        s.Trash(ctx, req),
		})
	}
	return results, nil
}

func requireReason(policy *LifecyclePolicy, reason string) error {
	if policy.RequireReason && reason == "" {
		return ErrReasonRequired
	}
	return nil
}

type auditInput struct {
	tenantID         string
	resourceType     string
	resourceID       uuid.UUID
	action           Action
	occurredAt       time.Time
	actor            string
	reason           string
	batchID          *uuid.UUID
	fromState        LifecycleState
	toState          LifecycleState
	retentionExpires *time.Time
}

func (s *Service) appendAudit(ctx context.Context, in auditInput) error {
	return s.audit.Append(ctx, LifecycleAudit{
		ID:               s.newID(),
		TenantID:         in.tenantID,
		ResourceType:     in.resourceType,
		ResourceID:       in.resourceID,
		Action:           in.action,
		OccurredAt:       in.occurredAt,
		Actor:            in.actor,
		Reason:           optionalReason(in.reason),
		BatchID:          in.batchID,
		FromState:        in.fromState,
		ToState:          in.toState,
		RetentionExpires: in.retentionExpires,
	})
}

func optionalReason(reason string) *string {
	if reason == "" {
		return nil
	}
	return &reason
}
