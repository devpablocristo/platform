package iam

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestValidateUserNormalizesExternalFields(t *testing.T) {
	t.Parallel()

	user, err := validateUser(User{
		Provider:      " clerk ",
		ExternalID:    " user_123 ",
		PrimaryEmail:  "  PERSON@Example.COM ",
		Name:          " Person ",
		AvatarURL:     " https://example.com/avatar.png ",
		Status:        UserActive,
		EmailVerified: true,
	})
	if err != nil {
		t.Fatalf("validateUser: %v", err)
	}
	if user.Provider != "clerk" || user.ExternalID != "user_123" ||
		user.PrimaryEmail != "person@example.com" || user.Name != "Person" {
		t.Fatalf("unexpected normalized user: %#v", user)
	}
}

func TestMembershipRemovalTimestampsAreConsistent(t *testing.T) {
	t.Parallel()

	base := Membership{
		OrganizationID: "5c270916-46bd-4daf-ad16-cebc5f51f058",
		UserID:         "bdef2743-8b5d-4236-bfde-937f5458cda2",
		Provider:       "provider",
		Role:           "opaque-role",
	}
	if _, err := validateMembership(withMembershipStatus(base, MembershipRemoved)); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("removed membership without timestamp error = %v", err)
	}

	now := time.Now().UTC()
	active := withMembershipStatus(base, MembershipActive)
	active.RemovedAt = &now
	if _, err := validateMembership(active); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("active membership with removed_at error = %v", err)
	}

	removed := withMembershipStatus(base, MembershipRemoved)
	removed.RemovedAt = &now
	if _, err := validateMembership(removed); err != nil {
		t.Fatalf("valid removed membership: %v", err)
	}
}

func TestInvitationTerminalTimestampsAreConsistent(t *testing.T) {
	t.Parallel()

	base := Invitation{
		OrganizationID: "5c270916-46bd-4daf-ad16-cebc5f51f058",
		Provider:       "provider",
		Email:          "person@example.com",
		Role:           "member",
		ExpiresAt:      time.Now().Add(time.Hour),
	}
	accepted := base
	accepted.Status = InvitationAccepted
	if _, err := validateInvitation(accepted); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("accepted invitation without timestamp error = %v", err)
	}

	now := time.Now().UTC()
	accepted.AcceptedAt = &now
	if _, err := validateInvitation(accepted); err != nil {
		t.Fatalf("valid accepted invitation: %v", err)
	}
}

func TestWebhookValidationCopiesPayload(t *testing.T) {
	t.Parallel()

	payload := json.RawMessage(`{"id":"evt_1"}`)
	event, err := validateWebhookEvent(WebhookEvent{
		Provider:   "provider",
		ExternalID: "evt_1",
		EventType:  "membership.created",
		Payload:    payload,
		OccurredAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("validateWebhookEvent: %v", err)
	}
	payload[2] = 'X'
	if string(event.Payload) != `{"id":"evt_1"}` {
		t.Fatalf("validated payload aliases caller memory: %s", event.Payload)
	}
}

func TestInvalidDomainValuesFailClosed(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		run  func() error
	}{
		{
			name: "organization status",
			run: func() error {
				_, err := validateOrganization(Organization{
					Provider: "provider", ExternalID: "org_1", Name: "Org", Status: "unknown",
				}, true)
				return err
			},
		},
		{
			name: "user identity",
			run: func() error {
				_, err := validateUser(User{Status: UserActive})
				return err
			},
		},
		{
			name: "invalid webhook JSON",
			run: func() error {
				_, err := validateWebhookEvent(WebhookEvent{
					Provider: "provider", ExternalID: "event", EventType: "type",
					Payload: json.RawMessage(`{`), OccurredAt: time.Now(),
				})
				return err
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if err := test.run(); !errors.Is(err, ErrInvalidArgument) {
				t.Fatalf("error = %v, want ErrInvalidArgument", err)
			}
		})
	}
}

func TestStoreErrorsAreStable(t *testing.T) {
	t.Parallel()

	if !errors.Is(storeError("read", pgx.ErrNoRows), ErrNotFound) {
		t.Fatal("pgx.ErrNoRows was not mapped to ErrNotFound")
	}
	if !errors.Is(storeError("write", &pgconn.PgError{
		Code: "23505", ConstraintName: "unique_identity",
	}), ErrConflict) {
		t.Fatal("unique violation was not mapped to ErrConflict")
	}
}

func TestNewPostgresStoreRejectsNilDatabase(t *testing.T) {
	t.Parallel()

	if _, err := NewPostgresStore(nil); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("NewPostgresStore(nil) error = %v", err)
	}
}

func withMembershipStatus(membership Membership, status MembershipStatus) Membership {
	membership.Status = status
	return membership
}
