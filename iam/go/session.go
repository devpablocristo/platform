package iam

import (
	"context"
	"fmt"
	"strings"
	"time"

	postgres "github.com/devpablocristo/platform/databases/postgres/go"
	"github.com/jackc/pgx/v5"
)

// VerifiedSession is a provider-neutral projection produced only after token
// signature and claim validation by an external verifier.
type VerifiedSession struct {
	Provider               string
	Subject                string
	SessionID              string
	ExternalOrganizationID string
	ProviderRole           string
	ProviderPermissions    []string
	IssuedAt               time.Time
	ExpiresAt              time.Time
}

// ValidateAt verifies the complete projection and its validity at now.
func (session VerifiedSession) ValidateAt(now time.Time) error {
	normalized, err := normalizeVerifiedSession(session)
	if err != nil {
		return err
	}
	if now.IsZero() {
		return fmt.Errorf("%w: current time is zero", ErrInvalidVerifiedSession)
	}
	now = now.UTC()
	if now.Before(normalized.IssuedAt) {
		return fmt.Errorf("%w: session is not active yet", ErrInvalidVerifiedSession)
	}
	if !now.Before(normalized.ExpiresAt) {
		return fmt.Errorf("%w: session expired", ErrInvalidVerifiedSession)
	}
	return nil
}

// ActiveMembership is the local authority resolved from a verified external
// subject and organization.
type ActiveMembership struct {
	MembershipID   string
	OrganizationID string
	UserID         string
	Role           string
}

// SessionTransactorConfig supplies deterministic runtime dependencies.
type SessionTransactorConfig struct {
	Now func() time.Time
}

// SessionTxFunc runs after local admission and transaction-scoped GUCs exist.
type SessionTxFunc func(context.Context, pgx.Tx, ActiveMembership) error

// SessionTransactor resolves local admission and scopes PostgreSQL access.
type SessionTransactor struct {
	unitOfWork *postgres.UnitOfWork[pgx.Tx]
	resolver   MembershipResolver
	now        func() time.Time
}

// NewSessionTransactor creates a fail-closed session transaction coordinator.
func NewSessionTransactor(
	beginner postgres.PgxBeginner,
	resolver MembershipResolver,
	config SessionTransactorConfig,
) (*SessionTransactor, error) {
	if beginner == nil {
		return nil, fmt.Errorf("%w: transaction beginner is nil", ErrInvalidArgument)
	}
	if resolver == nil {
		return nil, fmt.Errorf("%w: membership resolver is nil", ErrInvalidArgument)
	}
	if config.Now == nil {
		config.Now = time.Now
	}
	unitOfWork, err := postgres.NewPgxUnitOfWork(beginner)
	if err != nil {
		return nil, fmt.Errorf("create IAM unit of work: %w", err)
	}
	return &SessionTransactor{
		unitOfWork: unitOfWork,
		resolver:   resolver,
		now:        config.Now,
	}, nil
}

// WithinSessionTx validates session freshness, resolves an active local
// membership in the new transaction, applies app.user_id/app.org_id with
// transaction-local scope, and invokes fn. Errors and panics roll back through
// the PostgreSQL Unit of Work.
func (transactor *SessionTransactor) WithinSessionTx(
	ctx context.Context,
	session VerifiedSession,
	fn SessionTxFunc,
) error {
	if transactor == nil || transactor.unitOfWork == nil ||
		transactor.resolver == nil || transactor.now == nil {
		return fmt.Errorf("%w: session transactor is nil", ErrInvalidArgument)
	}
	if fn == nil {
		return fmt.Errorf("%w: session transaction callback is nil", ErrInvalidArgument)
	}
	normalized, err := normalizeVerifiedSession(session)
	if err != nil {
		return err
	}
	if err := normalized.ValidateAt(transactor.now()); err != nil {
		return err
	}

	return transactor.unitOfWork.WithinTx(ctx, func(txContext context.Context) error {
		tx, txErr := postgres.Tx[pgx.Tx](txContext)
		if txErr != nil {
			return fmt.Errorf("load IAM transaction: %w", txErr)
		}
		active, resolveErr := transactor.resolver.ResolveActiveMembership(
			txContext,
			tx,
			normalized,
		)
		if resolveErr != nil {
			return resolveErr
		}
		active, activeErr := normalizeActiveMembership(active)
		if activeErr != nil {
			return activeErr
		}
		if _, execErr := tx.Exec(txContext, `
			SELECT
				set_config('app.user_id', $1, true),
				set_config('app.org_id', $2, true)
		`, active.UserID, active.OrganizationID); execErr != nil {
			return fmt.Errorf("apply IAM transaction context: %w", execErr)
		}
		return fn(txContext, tx, active)
	})
}

func normalizeVerifiedSession(session VerifiedSession) (VerifiedSession, error) {
	session.Provider = strings.TrimSpace(session.Provider)
	session.Subject = strings.TrimSpace(session.Subject)
	session.SessionID = strings.TrimSpace(session.SessionID)
	session.ExternalOrganizationID = strings.TrimSpace(session.ExternalOrganizationID)
	session.ProviderRole = strings.TrimSpace(session.ProviderRole)
	if session.Provider == "" || session.Subject == "" || session.SessionID == "" ||
		session.ExternalOrganizationID == "" {
		return VerifiedSession{}, fmt.Errorf(
			"%w: provider, subject, session and organization are required",
			ErrInvalidVerifiedSession,
		)
	}
	if session.IssuedAt.IsZero() || session.ExpiresAt.IsZero() {
		return VerifiedSession{}, fmt.Errorf(
			"%w: issued and expiry times are required",
			ErrInvalidVerifiedSession,
		)
	}
	session.IssuedAt = session.IssuedAt.UTC()
	session.ExpiresAt = session.ExpiresAt.UTC()
	if !session.ExpiresAt.After(session.IssuedAt) {
		return VerifiedSession{}, fmt.Errorf(
			"%w: expiry must be after issue time",
			ErrInvalidVerifiedSession,
		)
	}
	session.ProviderPermissions = normalizeOpaqueValues(session.ProviderPermissions)
	return session, nil
}

func normalizeActiveMembership(active ActiveMembership) (ActiveMembership, error) {
	active.MembershipID = strings.TrimSpace(active.MembershipID)
	active.OrganizationID = strings.TrimSpace(active.OrganizationID)
	active.UserID = strings.TrimSpace(active.UserID)
	active.Role = strings.TrimSpace(active.Role)
	if active.MembershipID == "" || active.OrganizationID == "" ||
		active.UserID == "" || active.Role == "" {
		return ActiveMembership{}, fmt.Errorf(
			"%w: resolved membership is incomplete",
			ErrActiveMembershipRequired,
		)
	}
	return active, nil
}

func normalizeOpaqueValues(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	normalized := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}
	if len(normalized) == 0 {
		return nil
	}
	return normalized
}
