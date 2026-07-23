package iam

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// MembershipResolver maps a verified external identity to local admission.
// Implementations must return ErrActiveMembershipRequired for every
// non-admitted state without falling back to request metadata.
type MembershipResolver interface {
	ResolveActiveMembership(context.Context, DBTX, VerifiedSession) (ActiveMembership, error)
}

// MembershipResolverFunc adapts a function into MembershipResolver.
type MembershipResolverFunc func(
	context.Context,
	DBTX,
	VerifiedSession,
) (ActiveMembership, error)

// ResolveActiveMembership implements MembershipResolver.
func (function MembershipResolverFunc) ResolveActiveMembership(
	ctx context.Context,
	db DBTX,
	session VerifiedSession,
) (ActiveMembership, error) {
	if function == nil {
		return ActiveMembership{}, fmt.Errorf(
			"%w: membership resolver function is nil",
			ErrActiveMembershipRequired,
		)
	}
	return function(ctx, db, session)
}

// PostgresMembershipResolver resolves active records directly from iam tables.
// Consumers that protect bootstrap reads with RLS should provide a
// MembershipResolver backed by their narrowly scoped SECURITY DEFINER function.
type PostgresMembershipResolver struct{}

// NewPostgresMembershipResolver creates the direct PostgreSQL resolver.
func NewPostgresMembershipResolver() *PostgresMembershipResolver {
	return &PostgresMembershipResolver{}
}

// ResolveActiveMembership requires active user, organization and membership
// rows for the exact provider/subject/external organization tuple.
func (*PostgresMembershipResolver) ResolveActiveMembership(
	ctx context.Context,
	db DBTX,
	session VerifiedSession,
) (ActiveMembership, error) {
	if db == nil {
		return ActiveMembership{}, fmt.Errorf(
			"%w: resolver database is nil",
			ErrActiveMembershipRequired,
		)
	}
	normalized, err := normalizeVerifiedSession(session)
	if err != nil {
		return ActiveMembership{}, err
	}
	var active ActiveMembership
	err = db.QueryRow(ctx, `
		SELECT
			m.id::text,
			m.org_id::text,
			m.user_id::text,
			m.role
		FROM iam.memberships AS m
		JOIN iam.organizations AS o ON o.id = m.org_id
		JOIN iam.users AS u ON u.id = m.user_id
		WHERE u.provider = $1
		  AND u.external_id = $2
		  AND o.provider = $1
		  AND o.external_id = $3
		  AND u.status = 'active'
		  AND o.status = 'active'
		  AND m.status = 'active'
	`, normalized.Provider, normalized.Subject, normalized.ExternalOrganizationID).Scan(
		&active.MembershipID,
		&active.OrganizationID,
		&active.UserID,
		&active.Role,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return ActiveMembership{}, ErrActiveMembershipRequired
	}
	if err != nil {
		return ActiveMembership{}, fmt.Errorf("resolve active membership: %w", err)
	}
	return normalizeActiveMembership(active)
}
