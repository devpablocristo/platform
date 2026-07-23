package iam

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

// DBTX is implemented by pgxpool.Pool, pgx.Conn and pgx.Tx.
type DBTX interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

// PostgresStore persists provider-neutral IAM records.
type PostgresStore struct {
	db DBTX
}

// NewPostgresStore creates a store over a pool, connection or transaction.
func NewPostgresStore(db DBTX) (*PostgresStore, error) {
	if db == nil {
		return nil, fmt.Errorf("%w: database is nil", ErrInvalidArgument)
	}
	return &PostgresStore{db: db}, nil
}

// CreateOrganization inserts an organization. ID may be empty to let
// PostgreSQL generate it, and ExternalID may be empty while provisioning.
func (store *PostgresStore) CreateOrganization(
	ctx context.Context,
	organization Organization,
) (Organization, error) {
	if err := store.ready(); err != nil {
		return Organization{}, err
	}
	normalized, err := validateOrganization(organization, false)
	if err != nil {
		return Organization{}, err
	}
	row := store.db.QueryRow(ctx, `
		INSERT INTO iam.organizations (
			id, provider, external_id, name, slug, status, created_at, updated_at
		) VALUES (
			CASE WHEN $1 = '' THEN gen_random_uuid() ELSE $1::uuid END,
			$2, NULLIF($3, ''), $4, NULLIF($5, ''), $6, now(), now()
		)
		RETURNING `+organizationColumns,
		normalized.ID,
		normalized.Provider,
		normalized.ExternalID,
		normalized.Name,
		normalized.Slug,
		normalized.Status,
	)
	created, err := scanOrganization(row)
	if err != nil {
		return Organization{}, storeError("create organization", err)
	}
	return created, nil
}

// UpsertOrganization inserts or updates a provider organization by its stable
// external identity.
func (store *PostgresStore) UpsertOrganization(
	ctx context.Context,
	organization Organization,
) (Organization, error) {
	if err := store.ready(); err != nil {
		return Organization{}, err
	}
	normalized, err := validateOrganization(organization, true)
	if err != nil {
		return Organization{}, err
	}
	row := store.db.QueryRow(ctx, `
		INSERT INTO iam.organizations (
			id, provider, external_id, name, slug, status, created_at, updated_at
		) VALUES (
			CASE WHEN $1 = '' THEN gen_random_uuid() ELSE $1::uuid END,
			$2, $3, $4, NULLIF($5, ''), $6, now(), now()
		)
		ON CONFLICT (provider, external_id) WHERE external_id IS NOT NULL
		DO UPDATE SET
			name = EXCLUDED.name,
			slug = EXCLUDED.slug,
			status = EXCLUDED.status,
			updated_at = now()
		RETURNING `+organizationColumns,
		normalized.ID,
		normalized.Provider,
		normalized.ExternalID,
		normalized.Name,
		normalized.Slug,
		normalized.Status,
	)
	saved, err := scanOrganization(row)
	if err != nil {
		return Organization{}, storeError("upsert organization", err)
	}
	return saved, nil
}

// UpdateOrganization updates a locally provisioned organization, including
// attaching its external provider identity once one exists.
func (store *PostgresStore) UpdateOrganization(
	ctx context.Context,
	organization Organization,
) (Organization, error) {
	if err := store.ready(); err != nil {
		return Organization{}, err
	}
	normalized, err := validateOrganization(organization, false)
	if err != nil {
		return Organization{}, err
	}
	if normalized.ID == "" {
		return Organization{}, fmt.Errorf("%w: organization ID is required", ErrInvalidArgument)
	}
	row := store.db.QueryRow(ctx, `
		UPDATE iam.organizations
		SET provider = $2,
			external_id = NULLIF($3, ''),
			name = $4,
			slug = NULLIF($5, ''),
			status = $6,
			updated_at = now()
		WHERE id = $1::uuid
		RETURNING `+organizationColumns,
		normalized.ID,
		normalized.Provider,
		normalized.ExternalID,
		normalized.Name,
		normalized.Slug,
		normalized.Status,
	)
	updated, err := scanOrganization(row)
	if err != nil {
		return Organization{}, storeError("update organization", err)
	}
	return updated, nil
}

// GetOrganization returns an organization by local ID.
func (store *PostgresStore) GetOrganization(ctx context.Context, id string) (Organization, error) {
	if err := store.ready(); err != nil {
		return Organization{}, err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return Organization{}, fmt.Errorf("%w: organization ID is required", ErrInvalidArgument)
	}
	organization, err := scanOrganization(store.db.QueryRow(ctx, `
		SELECT `+organizationColumns+`
		FROM iam.organizations
		WHERE id = $1::uuid
	`, id))
	if err != nil {
		return Organization{}, storeError("get organization", err)
	}
	return organization, nil
}

// GetOrganizationByExternalID resolves an organization by provider identity.
func (store *PostgresStore) GetOrganizationByExternalID(
	ctx context.Context,
	provider string,
	externalID string,
) (Organization, error) {
	if err := store.ready(); err != nil {
		return Organization{}, err
	}
	provider, externalID, err := externalIdentity(provider, externalID)
	if err != nil {
		return Organization{}, err
	}
	organization, err := scanOrganization(store.db.QueryRow(ctx, `
		SELECT `+organizationColumns+`
		FROM iam.organizations
		WHERE provider = $1 AND external_id = $2
	`, provider, externalID))
	if err != nil {
		return Organization{}, storeError("get organization by external ID", err)
	}
	return organization, nil
}

// UpsertUser inserts or updates a user by provider identity.
func (store *PostgresStore) UpsertUser(ctx context.Context, user User) (User, error) {
	if err := store.ready(); err != nil {
		return User{}, err
	}
	normalized, err := validateUser(user)
	if err != nil {
		return User{}, err
	}
	row := store.db.QueryRow(ctx, `
		INSERT INTO iam.users (
			id, provider, external_id, primary_email, email_verified,
			name, avatar_url, status, created_at, updated_at
		) VALUES (
			CASE WHEN $1 = '' THEN gen_random_uuid() ELSE $1::uuid END,
			$2, $3, $4, $5, $6, NULLIF($7, ''), $8, now(), now()
		)
		ON CONFLICT (provider, external_id)
		DO UPDATE SET
			primary_email = EXCLUDED.primary_email,
			email_verified = EXCLUDED.email_verified,
			name = EXCLUDED.name,
			avatar_url = EXCLUDED.avatar_url,
			status = EXCLUDED.status,
			updated_at = now()
		RETURNING `+userColumns,
		normalized.ID,
		normalized.Provider,
		normalized.ExternalID,
		normalized.PrimaryEmail,
		normalized.EmailVerified,
		normalized.Name,
		normalized.AvatarURL,
		normalized.Status,
	)
	saved, err := scanUser(row)
	if err != nil {
		return User{}, storeError("upsert user", err)
	}
	return saved, nil
}

// GetUser returns a user by local ID.
func (store *PostgresStore) GetUser(ctx context.Context, id string) (User, error) {
	if err := store.ready(); err != nil {
		return User{}, err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return User{}, fmt.Errorf("%w: user ID is required", ErrInvalidArgument)
	}
	user, err := scanUser(store.db.QueryRow(ctx, `
		SELECT `+userColumns+`
		FROM iam.users
		WHERE id = $1::uuid
	`, id))
	if err != nil {
		return User{}, storeError("get user", err)
	}
	return user, nil
}

// GetUserByExternalID resolves a user by provider identity.
func (store *PostgresStore) GetUserByExternalID(
	ctx context.Context,
	provider string,
	externalID string,
) (User, error) {
	if err := store.ready(); err != nil {
		return User{}, err
	}
	provider, externalID, err := externalIdentity(provider, externalID)
	if err != nil {
		return User{}, err
	}
	user, err := scanUser(store.db.QueryRow(ctx, `
		SELECT `+userColumns+`
		FROM iam.users
		WHERE provider = $1 AND external_id = $2
	`, provider, externalID))
	if err != nil {
		return User{}, storeError("get user by external ID", err)
	}
	return user, nil
}

// UpsertMembership inserts or updates the single membership for an
// organization/user pair.
func (store *PostgresStore) UpsertMembership(
	ctx context.Context,
	membership Membership,
) (Membership, error) {
	if err := store.ready(); err != nil {
		return Membership{}, err
	}
	normalized, err := validateMembership(membership)
	if err != nil {
		return Membership{}, err
	}
	row := store.db.QueryRow(ctx, `
		INSERT INTO iam.memberships (
			id, org_id, user_id, provider, external_id, role, status,
			joined_at, removed_at, created_at, updated_at
		) VALUES (
			CASE WHEN $1 = '' THEN gen_random_uuid() ELSE $1::uuid END,
			$2::uuid, $3::uuid, $4, NULLIF($5, ''), $6, $7,
			$8, $9, now(), now()
		)
		ON CONFLICT (org_id, user_id)
		DO UPDATE SET
			provider = EXCLUDED.provider,
			external_id = EXCLUDED.external_id,
			role = EXCLUDED.role,
			status = EXCLUDED.status,
			joined_at = COALESCE(EXCLUDED.joined_at, iam.memberships.joined_at),
			removed_at = EXCLUDED.removed_at,
			updated_at = now()
		RETURNING `+membershipColumns,
		normalized.ID,
		normalized.OrganizationID,
		normalized.UserID,
		normalized.Provider,
		normalized.ExternalID,
		normalized.Role,
		normalized.Status,
		normalized.JoinedAt,
		normalized.RemovedAt,
	)
	saved, err := scanMembership(row)
	if err != nil {
		return Membership{}, storeError("upsert membership", err)
	}
	return saved, nil
}

// GetMembership returns a membership by local ID.
func (store *PostgresStore) GetMembership(ctx context.Context, id string) (Membership, error) {
	if err := store.ready(); err != nil {
		return Membership{}, err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return Membership{}, fmt.Errorf("%w: membership ID is required", ErrInvalidArgument)
	}
	membership, err := scanMembership(store.db.QueryRow(ctx, `
		SELECT `+membershipColumns+`
		FROM iam.memberships
		WHERE id = $1::uuid
	`, id))
	if err != nil {
		return Membership{}, storeError("get membership", err)
	}
	return membership, nil
}

// ListMembershipsByOrganization returns all local membership states.
func (store *PostgresStore) ListMembershipsByOrganization(
	ctx context.Context,
	organizationID string,
) ([]Membership, error) {
	if err := store.ready(); err != nil {
		return nil, err
	}
	organizationID = strings.TrimSpace(organizationID)
	if organizationID == "" {
		return nil, fmt.Errorf("%w: organization ID is required", ErrInvalidArgument)
	}
	rows, err := store.db.Query(ctx, `
		SELECT `+membershipColumns+`
		FROM iam.memberships
		WHERE org_id = $1::uuid
		ORDER BY created_at, id
	`, organizationID)
	if err != nil {
		return nil, storeError("list organization memberships", err)
	}
	defer rows.Close()

	memberships := make([]Membership, 0)
	for rows.Next() {
		membership, scanErr := scanMembership(rows)
		if scanErr != nil {
			return nil, storeError("scan organization membership", scanErr)
		}
		memberships = append(memberships, membership)
	}
	if err := rows.Err(); err != nil {
		return nil, storeError("iterate organization memberships", err)
	}
	return memberships, nil
}

// ListActiveOrganizationsForUser returns only locally admitted memberships
// whose user and organization are also active.
func (store *PostgresStore) ListActiveOrganizationsForUser(
	ctx context.Context,
	userID string,
) ([]OrganizationAccess, error) {
	if err := store.ready(); err != nil {
		return nil, err
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, fmt.Errorf("%w: user ID is required", ErrInvalidArgument)
	}
	rows, err := store.db.Query(ctx, `
		SELECT
			o.id::text, o.provider, COALESCE(o.external_id, ''), o.name,
			COALESCE(o.slug, ''), o.status, o.created_at, o.updated_at,
			m.id::text, m.org_id::text, m.user_id::text, m.provider,
			COALESCE(m.external_id, ''), m.role, m.status, m.joined_at,
			m.removed_at, m.created_at, m.updated_at
		FROM iam.memberships AS m
		JOIN iam.organizations AS o ON o.id = m.org_id
		JOIN iam.users AS u ON u.id = m.user_id
		WHERE m.user_id = $1::uuid
		  AND m.status = 'active'
		  AND o.status = 'active'
		  AND u.status = 'active'
		ORDER BY lower(o.name), o.id
	`, userID)
	if err != nil {
		return nil, storeError("list user organizations", err)
	}
	defer rows.Close()

	accesses := make([]OrganizationAccess, 0)
	for rows.Next() {
		access, scanErr := scanOrganizationAccess(rows)
		if scanErr != nil {
			return nil, storeError("scan user organization", scanErr)
		}
		accesses = append(accesses, access)
	}
	if err := rows.Err(); err != nil {
		return nil, storeError("iterate user organizations", err)
	}
	return accesses, nil
}

// CreateInvitation inserts a local invitation projection.
func (store *PostgresStore) CreateInvitation(
	ctx context.Context,
	invitation Invitation,
) (Invitation, error) {
	if err := store.ready(); err != nil {
		return Invitation{}, err
	}
	normalized, err := validateInvitation(invitation)
	if err != nil {
		return Invitation{}, err
	}
	row := store.db.QueryRow(ctx, `
		INSERT INTO iam.invitations (
			id, org_id, provider, external_id, email_normalized, role, status,
			expires_at, accepted_at, revoked_at, created_at, updated_at
		) VALUES (
			CASE WHEN $1 = '' THEN gen_random_uuid() ELSE $1::uuid END,
			$2::uuid, $3, NULLIF($4, ''), $5, $6, $7,
			$8, $9, $10, now(), now()
		)
		RETURNING `+invitationColumns,
		normalized.ID,
		normalized.OrganizationID,
		normalized.Provider,
		normalized.ExternalID,
		normalized.Email,
		normalized.Role,
		normalized.Status,
		normalized.ExpiresAt,
		normalized.AcceptedAt,
		normalized.RevokedAt,
	)
	created, err := scanInvitation(row)
	if err != nil {
		return Invitation{}, storeError("create invitation", err)
	}
	return created, nil
}

// UpdateInvitation replaces the mutable projection fields of an invitation.
func (store *PostgresStore) UpdateInvitation(
	ctx context.Context,
	invitation Invitation,
) (Invitation, error) {
	if err := store.ready(); err != nil {
		return Invitation{}, err
	}
	normalized, err := validateInvitation(invitation)
	if err != nil {
		return Invitation{}, err
	}
	if normalized.ID == "" {
		return Invitation{}, fmt.Errorf("%w: invitation ID is required", ErrInvalidArgument)
	}
	row := store.db.QueryRow(ctx, `
		UPDATE iam.invitations
		SET provider = $2,
			external_id = NULLIF($3, ''),
			email_normalized = $4,
			role = $5,
			status = $6,
			expires_at = $7,
			accepted_at = $8,
			revoked_at = $9,
			updated_at = now()
		WHERE id = $1::uuid
		RETURNING `+invitationColumns,
		normalized.ID,
		normalized.Provider,
		normalized.ExternalID,
		normalized.Email,
		normalized.Role,
		normalized.Status,
		normalized.ExpiresAt,
		normalized.AcceptedAt,
		normalized.RevokedAt,
	)
	updated, err := scanInvitation(row)
	if err != nil {
		return Invitation{}, storeError("update invitation", err)
	}
	return updated, nil
}

// GetInvitation returns an invitation by local ID.
func (store *PostgresStore) GetInvitation(ctx context.Context, id string) (Invitation, error) {
	if err := store.ready(); err != nil {
		return Invitation{}, err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return Invitation{}, fmt.Errorf("%w: invitation ID is required", ErrInvalidArgument)
	}
	invitation, err := scanInvitation(store.db.QueryRow(ctx, `
		SELECT `+invitationColumns+`
		FROM iam.invitations
		WHERE id = $1::uuid
	`, id))
	if err != nil {
		return Invitation{}, storeError("get invitation", err)
	}
	return invitation, nil
}

// ListInvitationsByOrganization returns all invitation states newest first.
func (store *PostgresStore) ListInvitationsByOrganization(
	ctx context.Context,
	organizationID string,
) ([]Invitation, error) {
	if err := store.ready(); err != nil {
		return nil, err
	}
	organizationID = strings.TrimSpace(organizationID)
	if organizationID == "" {
		return nil, fmt.Errorf("%w: organization ID is required", ErrInvalidArgument)
	}
	rows, err := store.db.Query(ctx, `
		SELECT `+invitationColumns+`
		FROM iam.invitations
		WHERE org_id = $1::uuid
		ORDER BY created_at DESC, id
	`, organizationID)
	if err != nil {
		return nil, storeError("list organization invitations", err)
	}
	defer rows.Close()

	invitations := make([]Invitation, 0)
	for rows.Next() {
		invitation, scanErr := scanInvitation(rows)
		if scanErr != nil {
			return nil, storeError("scan organization invitation", scanErr)
		}
		invitations = append(invitations, invitation)
	}
	if err := rows.Err(); err != nil {
		return nil, storeError("iterate organization invitations", err)
	}
	return invitations, nil
}

// ReceiveWebhookEvent persists a provider event exactly once. created is false
// when the same provider/external ID was already recorded.
func (store *PostgresStore) ReceiveWebhookEvent(
	ctx context.Context,
	event WebhookEvent,
) (stored WebhookEvent, created bool, err error) {
	if readyErr := store.ready(); readyErr != nil {
		return WebhookEvent{}, false, readyErr
	}
	normalized, err := validateWebhookEvent(event)
	if err != nil {
		return WebhookEvent{}, false, err
	}
	row := store.db.QueryRow(ctx, `
		INSERT INTO iam.webhook_events (
			id, provider, external_id, event_type, payload, occurred_at,
			status, attempts, received_at, updated_at
		) VALUES (
			CASE WHEN $1 = '' THEN gen_random_uuid() ELSE $1::uuid END,
			$2, $3, $4, $5::jsonb, $6, 'pending', 0, now(), now()
		)
		ON CONFLICT (provider, external_id) DO NOTHING
		RETURNING `+webhookEventColumns,
		normalized.ID,
		normalized.Provider,
		normalized.ExternalID,
		normalized.EventType,
		[]byte(normalized.Payload),
		normalized.OccurredAt,
	)
	stored, err = scanWebhookEvent(row)
	if err == nil {
		return stored, true, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return WebhookEvent{}, false, storeError("receive webhook event", err)
	}
	stored, err = store.GetWebhookEvent(ctx, normalized.Provider, normalized.ExternalID)
	if err != nil {
		return WebhookEvent{}, false, err
	}
	return stored, false, nil
}

// GetWebhookEvent resolves a provider inbox event.
func (store *PostgresStore) GetWebhookEvent(
	ctx context.Context,
	provider string,
	externalID string,
) (WebhookEvent, error) {
	if err := store.ready(); err != nil {
		return WebhookEvent{}, err
	}
	provider, externalID, err := externalIdentity(provider, externalID)
	if err != nil {
		return WebhookEvent{}, err
	}
	event, err := scanWebhookEvent(store.db.QueryRow(ctx, `
		SELECT `+webhookEventColumns+`
		FROM iam.webhook_events
		WHERE provider = $1 AND external_id = $2
	`, provider, externalID))
	if err != nil {
		return WebhookEvent{}, storeError("get webhook event", err)
	}
	return event, nil
}

// MarkWebhookEventProcessed records a successful processing attempt.
func (store *PostgresStore) MarkWebhookEventProcessed(
	ctx context.Context,
	provider string,
	externalID string,
) (WebhookEvent, error) {
	if err := store.ready(); err != nil {
		return WebhookEvent{}, err
	}
	provider, externalID, err := externalIdentity(provider, externalID)
	if err != nil {
		return WebhookEvent{}, err
	}
	event, err := scanWebhookEvent(store.db.QueryRow(ctx, `
		UPDATE iam.webhook_events
		SET status = 'processed',
			attempts = attempts + 1,
			processed_at = now(),
			last_error = NULL,
			updated_at = now()
		WHERE provider = $1 AND external_id = $2
		RETURNING `+webhookEventColumns,
		provider,
		externalID,
	))
	if err != nil {
		return WebhookEvent{}, storeError("mark webhook event processed", err)
	}
	return event, nil
}

// MarkWebhookEventFailed records a failed processing attempt. A later
// successful attempt may transition the event to processed.
func (store *PostgresStore) MarkWebhookEventFailed(
	ctx context.Context,
	provider string,
	externalID string,
	failure error,
) (WebhookEvent, error) {
	if err := store.ready(); err != nil {
		return WebhookEvent{}, err
	}
	provider, externalID, err := externalIdentity(provider, externalID)
	if err != nil {
		return WebhookEvent{}, err
	}
	if failure == nil || strings.TrimSpace(failure.Error()) == "" {
		return WebhookEvent{}, fmt.Errorf("%w: webhook failure is required", ErrInvalidArgument)
	}
	event, err := scanWebhookEvent(store.db.QueryRow(ctx, `
		UPDATE iam.webhook_events
		SET status = 'failed',
			attempts = attempts + 1,
			processed_at = NULL,
			last_error = $3,
			updated_at = now()
		WHERE provider = $1 AND external_id = $2
		RETURNING `+webhookEventColumns,
		provider,
		externalID,
		failure.Error(),
	))
	if err != nil {
		return WebhookEvent{}, storeError("mark webhook event failed", err)
	}
	return event, nil
}

func (store *PostgresStore) ready() error {
	if store == nil || store.db == nil {
		return fmt.Errorf("%w: postgres store is nil", ErrInvalidArgument)
	}
	return nil
}

func externalIdentity(provider string, externalID string) (string, string, error) {
	provider = strings.TrimSpace(provider)
	externalID = strings.TrimSpace(externalID)
	if provider == "" || externalID == "" {
		return "", "", fmt.Errorf("%w: provider and external ID are required", ErrInvalidArgument)
	}
	return provider, externalID, nil
}

const organizationColumns = `id::text, provider, COALESCE(external_id, ''), name,
	COALESCE(slug, ''), status, created_at, updated_at`

const userColumns = `id::text, provider, external_id, primary_email,
	email_verified, name, COALESCE(avatar_url, ''), status, created_at, updated_at`

const membershipColumns = `id::text, org_id::text, user_id::text, provider,
	COALESCE(external_id, ''), role, status, joined_at, removed_at, created_at, updated_at`

const invitationColumns = `id::text, org_id::text, provider,
	COALESCE(external_id, ''), email_normalized, role, status, expires_at,
	accepted_at, revoked_at, created_at, updated_at`

const webhookEventColumns = `id::text, provider, external_id, event_type,
	payload, occurred_at, status, attempts, processed_at, COALESCE(last_error, ''),
	received_at, updated_at`

type rowScanner interface {
	Scan(...any) error
}

func scanOrganization(row rowScanner) (Organization, error) {
	var organization Organization
	err := row.Scan(
		&organization.ID,
		&organization.Provider,
		&organization.ExternalID,
		&organization.Name,
		&organization.Slug,
		&organization.Status,
		&organization.CreatedAt,
		&organization.UpdatedAt,
	)
	return organization, err
}

func scanUser(row rowScanner) (User, error) {
	var user User
	err := row.Scan(
		&user.ID,
		&user.Provider,
		&user.ExternalID,
		&user.PrimaryEmail,
		&user.EmailVerified,
		&user.Name,
		&user.AvatarURL,
		&user.Status,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	return user, err
}

func scanMembership(row rowScanner) (Membership, error) {
	var (
		membership Membership
		joinedAt   pgtype.Timestamptz
		removedAt  pgtype.Timestamptz
	)
	err := row.Scan(
		&membership.ID,
		&membership.OrganizationID,
		&membership.UserID,
		&membership.Provider,
		&membership.ExternalID,
		&membership.Role,
		&membership.Status,
		&joinedAt,
		&removedAt,
		&membership.CreatedAt,
		&membership.UpdatedAt,
	)
	if err != nil {
		return Membership{}, err
	}
	membership.JoinedAt = nullableTime(joinedAt)
	membership.RemovedAt = nullableTime(removedAt)
	return membership, nil
}

func scanInvitation(row rowScanner) (Invitation, error) {
	var (
		invitation Invitation
		acceptedAt pgtype.Timestamptz
		revokedAt  pgtype.Timestamptz
	)
	err := row.Scan(
		&invitation.ID,
		&invitation.OrganizationID,
		&invitation.Provider,
		&invitation.ExternalID,
		&invitation.Email,
		&invitation.Role,
		&invitation.Status,
		&invitation.ExpiresAt,
		&acceptedAt,
		&revokedAt,
		&invitation.CreatedAt,
		&invitation.UpdatedAt,
	)
	if err != nil {
		return Invitation{}, err
	}
	invitation.AcceptedAt = nullableTime(acceptedAt)
	invitation.RevokedAt = nullableTime(revokedAt)
	return invitation, nil
}

func scanWebhookEvent(row rowScanner) (WebhookEvent, error) {
	var (
		event       WebhookEvent
		payload     []byte
		processedAt pgtype.Timestamptz
	)
	err := row.Scan(
		&event.ID,
		&event.Provider,
		&event.ExternalID,
		&event.EventType,
		&payload,
		&event.OccurredAt,
		&event.Status,
		&event.Attempts,
		&processedAt,
		&event.LastError,
		&event.ReceivedAt,
		&event.UpdatedAt,
	)
	if err != nil {
		return WebhookEvent{}, err
	}
	event.Payload = append(json.RawMessage(nil), payload...)
	event.ProcessedAt = nullableTime(processedAt)
	return event, nil
}

func scanOrganizationAccess(row rowScanner) (OrganizationAccess, error) {
	var (
		access    OrganizationAccess
		joinedAt  pgtype.Timestamptz
		removedAt pgtype.Timestamptz
	)
	err := row.Scan(
		&access.Organization.ID,
		&access.Organization.Provider,
		&access.Organization.ExternalID,
		&access.Organization.Name,
		&access.Organization.Slug,
		&access.Organization.Status,
		&access.Organization.CreatedAt,
		&access.Organization.UpdatedAt,
		&access.Membership.ID,
		&access.Membership.OrganizationID,
		&access.Membership.UserID,
		&access.Membership.Provider,
		&access.Membership.ExternalID,
		&access.Membership.Role,
		&access.Membership.Status,
		&joinedAt,
		&removedAt,
		&access.Membership.CreatedAt,
		&access.Membership.UpdatedAt,
	)
	if err != nil {
		return OrganizationAccess{}, err
	}
	access.Membership.JoinedAt = nullableTime(joinedAt)
	access.Membership.RemovedAt = nullableTime(removedAt)
	return access, nil
}

func nullableTime(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	normalized := value.Time.UTC()
	return &normalized
}
