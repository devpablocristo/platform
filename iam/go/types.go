package iam

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// OrganizationStatus is the provider-independent lifecycle of an organization.
type OrganizationStatus string

const (
	OrganizationProvisioning OrganizationStatus = "provisioning"
	OrganizationActive       OrganizationStatus = "active"
	OrganizationSuspended    OrganizationStatus = "suspended"
	OrganizationDisabled     OrganizationStatus = "disabled"
)

// UserStatus is the local admission state of an external user.
type UserStatus string

const (
	UserActive   UserStatus = "active"
	UserDisabled UserStatus = "disabled"
)

// MembershipStatus is the local lifecycle of organization access.
type MembershipStatus string

const (
	MembershipPending     MembershipStatus = "pending"
	MembershipActive      MembershipStatus = "active"
	MembershipRemoved     MembershipStatus = "removed"
	MembershipQuarantined MembershipStatus = "quarantined"
)

// InvitationStatus is the local lifecycle of a provider invitation.
type InvitationStatus string

const (
	InvitationPending  InvitationStatus = "pending"
	InvitationAccepted InvitationStatus = "accepted"
	InvitationRevoked  InvitationStatus = "revoked"
	InvitationExpired  InvitationStatus = "expired"
)

// WebhookEventStatus is the processing state of an inbox event.
type WebhookEventStatus string

const (
	WebhookEventPending   WebhookEventStatus = "pending"
	WebhookEventProcessed WebhookEventStatus = "processed"
	WebhookEventFailed    WebhookEventStatus = "failed"
)

// Organization maps a provider organization to a local stable identifier.
type Organization struct {
	ID         string
	Provider   string
	ExternalID string
	Name       string
	Slug       string
	Status     OrganizationStatus
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// User maps a provider user to a local stable identifier.
type User struct {
	ID            string
	Provider      string
	ExternalID    string
	PrimaryEmail  string
	EmailVerified bool
	Name          string
	AvatarURL     string
	Status        UserStatus
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// Membership is the local authority for a user's organization access.
// Role is intentionally opaque: products own their role vocabulary.
type Membership struct {
	ID             string
	OrganizationID string
	UserID         string
	Provider       string
	ExternalID     string
	Role           string
	Status         MembershipStatus
	JoinedAt       *time.Time
	RemovedAt      *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// Invitation records the local projection of an external invitation.
type Invitation struct {
	ID             string
	OrganizationID string
	Provider       string
	ExternalID     string
	Email          string
	Role           string
	Status         InvitationStatus
	ExpiresAt      time.Time
	AcceptedAt     *time.Time
	RevokedAt      *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// WebhookEvent is a deduplicated provider event stored for explicit processing.
type WebhookEvent struct {
	ID          string
	Provider    string
	ExternalID  string
	EventType   string
	Payload     json.RawMessage
	OccurredAt  time.Time
	Status      WebhookEventStatus
	Attempts    int
	ProcessedAt *time.Time
	LastError   string
	ReceivedAt  time.Time
	UpdatedAt   time.Time
}

// OrganizationAccess combines an active organization and membership.
type OrganizationAccess struct {
	Organization Organization
	Membership   Membership
}

func validateOrganization(organization Organization, externalIDRequired bool) (Organization, error) {
	organization.ID = strings.TrimSpace(organization.ID)
	organization.Provider = strings.TrimSpace(organization.Provider)
	organization.ExternalID = strings.TrimSpace(organization.ExternalID)
	organization.Name = strings.TrimSpace(organization.Name)
	organization.Slug = strings.TrimSpace(organization.Slug)
	if organization.Provider == "" || organization.Name == "" {
		return Organization{}, fmt.Errorf("%w: organization provider and name are required", ErrInvalidArgument)
	}
	if externalIDRequired && organization.ExternalID == "" {
		return Organization{}, fmt.Errorf("%w: organization external ID is required", ErrInvalidArgument)
	}
	if !organization.Status.valid() {
		return Organization{}, fmt.Errorf("%w: unsupported organization status %q", ErrInvalidArgument, organization.Status)
	}
	return organization, nil
}

func validateUser(user User) (User, error) {
	user.ID = strings.TrimSpace(user.ID)
	user.Provider = strings.TrimSpace(user.Provider)
	user.ExternalID = strings.TrimSpace(user.ExternalID)
	user.PrimaryEmail = normalizeEmail(user.PrimaryEmail)
	user.Name = strings.TrimSpace(user.Name)
	user.AvatarURL = strings.TrimSpace(user.AvatarURL)
	if user.Provider == "" || user.ExternalID == "" || user.PrimaryEmail == "" {
		return User{}, fmt.Errorf("%w: user provider, external ID and primary email are required", ErrInvalidArgument)
	}
	if !user.Status.valid() {
		return User{}, fmt.Errorf("%w: unsupported user status %q", ErrInvalidArgument, user.Status)
	}
	return user, nil
}

func validateMembership(membership Membership) (Membership, error) {
	membership.ID = strings.TrimSpace(membership.ID)
	membership.OrganizationID = strings.TrimSpace(membership.OrganizationID)
	membership.UserID = strings.TrimSpace(membership.UserID)
	membership.Provider = strings.TrimSpace(membership.Provider)
	membership.ExternalID = strings.TrimSpace(membership.ExternalID)
	membership.Role = strings.TrimSpace(membership.Role)
	if membership.OrganizationID == "" || membership.UserID == "" ||
		membership.Provider == "" || membership.Role == "" {
		return Membership{}, fmt.Errorf("%w: membership organization, user, provider and role are required", ErrInvalidArgument)
	}
	if !membership.Status.valid() {
		return Membership{}, fmt.Errorf("%w: unsupported membership status %q", ErrInvalidArgument, membership.Status)
	}
	if membership.Status == MembershipRemoved && membership.RemovedAt == nil {
		return Membership{}, fmt.Errorf("%w: removed membership requires removed_at", ErrInvalidArgument)
	}
	if membership.Status != MembershipRemoved && membership.RemovedAt != nil {
		return Membership{}, fmt.Errorf("%w: only removed membership may set removed_at", ErrInvalidArgument)
	}
	return membership, nil
}

func validateInvitation(invitation Invitation) (Invitation, error) {
	invitation.ID = strings.TrimSpace(invitation.ID)
	invitation.OrganizationID = strings.TrimSpace(invitation.OrganizationID)
	invitation.Provider = strings.TrimSpace(invitation.Provider)
	invitation.ExternalID = strings.TrimSpace(invitation.ExternalID)
	invitation.Email = normalizeEmail(invitation.Email)
	invitation.Role = strings.TrimSpace(invitation.Role)
	if invitation.OrganizationID == "" || invitation.Provider == "" ||
		invitation.Email == "" || invitation.Role == "" || invitation.ExpiresAt.IsZero() {
		return Invitation{}, fmt.Errorf("%w: invitation organization, provider, email, role and expiry are required", ErrInvalidArgument)
	}
	if !invitation.Status.valid() {
		return Invitation{}, fmt.Errorf("%w: unsupported invitation status %q", ErrInvalidArgument, invitation.Status)
	}
	if invitation.Status == InvitationAccepted && invitation.AcceptedAt == nil {
		return Invitation{}, fmt.Errorf("%w: accepted invitation requires accepted_at", ErrInvalidArgument)
	}
	if invitation.Status != InvitationAccepted && invitation.AcceptedAt != nil {
		return Invitation{}, fmt.Errorf("%w: only accepted invitation may set accepted_at", ErrInvalidArgument)
	}
	if invitation.Status == InvitationRevoked && invitation.RevokedAt == nil {
		return Invitation{}, fmt.Errorf("%w: revoked invitation requires revoked_at", ErrInvalidArgument)
	}
	if invitation.Status != InvitationRevoked && invitation.RevokedAt != nil {
		return Invitation{}, fmt.Errorf("%w: only revoked invitation may set revoked_at", ErrInvalidArgument)
	}
	invitation.ExpiresAt = invitation.ExpiresAt.UTC()
	return invitation, nil
}

func validateWebhookEvent(event WebhookEvent) (WebhookEvent, error) {
	event.ID = strings.TrimSpace(event.ID)
	event.Provider = strings.TrimSpace(event.Provider)
	event.ExternalID = strings.TrimSpace(event.ExternalID)
	event.EventType = strings.TrimSpace(event.EventType)
	if event.Provider == "" || event.ExternalID == "" || event.EventType == "" {
		return WebhookEvent{}, fmt.Errorf("%w: webhook provider, external ID and event type are required", ErrInvalidArgument)
	}
	if len(event.Payload) == 0 || !json.Valid(event.Payload) {
		return WebhookEvent{}, fmt.Errorf("%w: webhook payload must be valid JSON", ErrInvalidArgument)
	}
	if event.OccurredAt.IsZero() {
		return WebhookEvent{}, fmt.Errorf("%w: webhook occurrence time is required", ErrInvalidArgument)
	}
	event.OccurredAt = event.OccurredAt.UTC()
	event.Payload = append(json.RawMessage(nil), event.Payload...)
	return event, nil
}

func normalizeEmail(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func (status OrganizationStatus) valid() bool {
	switch status {
	case OrganizationProvisioning, OrganizationActive, OrganizationSuspended, OrganizationDisabled:
		return true
	default:
		return false
	}
}

func (status UserStatus) valid() bool {
	return status == UserActive || status == UserDisabled
}

func (status MembershipStatus) valid() bool {
	switch status {
	case MembershipPending, MembershipActive, MembershipRemoved, MembershipQuarantined:
		return true
	default:
		return false
	}
}

func (status InvitationStatus) valid() bool {
	switch status {
	case InvitationPending, InvitationAccepted, InvitationRevoked, InvitationExpired:
		return true
	default:
		return false
	}
}
