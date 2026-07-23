package iam

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func iamIntegrationPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv("PLATFORM_POSTGRES_TEST_DSN"))
	if dsn == "" {
		if strings.EqualFold(os.Getenv("CI"), "true") {
			t.Fatal("PLATFORM_POSTGRES_TEST_DSN is required in CI")
		}
		t.Skip("PLATFORM_POSTGRES_TEST_DSN is not set")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect PostgreSQL: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping PostgreSQL: %v", err)
	}
	migration, err := fs.ReadFile(Migrations, MigrationsDir+"/0001_iam_core.sql")
	if err != nil {
		t.Fatalf("read IAM migration: %v", err)
	}
	for run := 0; run < 2; run++ {
		if _, err := pool.Exec(ctx, string(migration)); err != nil {
			t.Fatalf("apply IAM migration run %d: %v", run+1, err)
		}
	}
	return pool
}

func TestPostgresStoresRoundTripAndDeduplicate(t *testing.T) {
	pool := iamIntegrationPool(t)
	store, err := NewPostgresStore(pool)
	if err != nil {
		t.Fatal(err)
	}
	token := fmt.Sprintf("%d", time.Now().UnixNano())
	provider := "iam-integration-" + token
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM iam.webhook_events WHERE provider = $1", provider)
		_, _ = pool.Exec(context.Background(), "DELETE FROM iam.organizations WHERE provider = $1", provider)
		_, _ = pool.Exec(context.Background(), "DELETE FROM iam.users WHERE provider = $1", provider)
	})

	organization, err := store.UpsertOrganization(t.Context(), Organization{
		Provider: provider, ExternalID: "org_external", Name: "Initial",
		Slug: "org-" + token, Status: OrganizationActive,
	})
	if err != nil {
		t.Fatalf("upsert organization: %v", err)
	}
	updatedOrganization, err := store.UpsertOrganization(t.Context(), Organization{
		Provider: provider, ExternalID: "org_external", Name: "Updated",
		Slug: "org-" + token, Status: OrganizationActive,
	})
	if err != nil {
		t.Fatalf("update organization: %v", err)
	}
	if updatedOrganization.ID != organization.ID || updatedOrganization.Name != "Updated" {
		t.Fatalf("organization upsert was not stable: before=%#v after=%#v", organization, updatedOrganization)
	}
	provisioning, err := store.CreateOrganization(t.Context(), Organization{
		Provider: provider, Name: "Provisioning", Slug: "provisioning-" + token,
		Status: OrganizationProvisioning,
	})
	if err != nil {
		t.Fatalf("create provisioning organization: %v", err)
	}
	provisioning.ExternalID = "provisioned_external"
	provisioning.Status = OrganizationActive
	provisioned, err := store.UpdateOrganization(t.Context(), provisioning)
	if err != nil {
		t.Fatalf("attach organization external identity: %v", err)
	}
	if provisioned.ID != provisioning.ID || provisioned.ExternalID != "provisioned_external" {
		t.Fatalf("unexpected provisioned organization: %#v", provisioned)
	}

	user, err := store.UpsertUser(t.Context(), User{
		Provider: provider, ExternalID: "user_external", PrimaryEmail: " PERSON@EXAMPLE.COM ",
		EmailVerified: true, Name: "Person", Status: UserActive,
	})
	if err != nil {
		t.Fatalf("upsert user: %v", err)
	}
	if user.PrimaryEmail != "person@example.com" {
		t.Fatalf("email was not normalized: %q", user.PrimaryEmail)
	}

	joinedAt := time.Now().UTC().Truncate(time.Microsecond)
	membership, err := store.UpsertMembership(t.Context(), Membership{
		OrganizationID: organization.ID,
		UserID:         user.ID,
		Provider:       provider,
		ExternalID:     "membership_external",
		Role:           "consumer-defined-role",
		Status:         MembershipActive,
		JoinedAt:       &joinedAt,
	})
	if err != nil {
		t.Fatalf("upsert membership: %v", err)
	}
	accesses, err := store.ListActiveOrganizationsForUser(t.Context(), user.ID)
	if err != nil {
		t.Fatalf("list active organizations: %v", err)
	}
	if len(accesses) != 1 || accesses[0].Membership.ID != membership.ID ||
		accesses[0].Organization.ID != organization.ID {
		t.Fatalf("unexpected accesses: %#v", accesses)
	}

	invitation, err := store.CreateInvitation(t.Context(), Invitation{
		OrganizationID: organization.ID,
		Provider:       provider,
		ExternalID:     "invitation_external",
		Email:          " INVITED@EXAMPLE.COM ",
		Role:           "consumer-defined-role",
		Status:         InvitationPending,
		ExpiresAt:      time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("create invitation: %v", err)
	}
	if invitation.Email != "invited@example.com" {
		t.Fatalf("invitation email was not normalized: %q", invitation.Email)
	}
	if _, err := store.CreateInvitation(t.Context(), Invitation{
		OrganizationID: organization.ID,
		Provider:       provider,
		Email:          invitation.Email,
		Role:           invitation.Role,
		Status:         InvitationPending,
		ExpiresAt:      time.Now().Add(time.Hour),
	}); !errors.Is(err, ErrConflict) {
		t.Fatalf("duplicate pending invitation error = %v, want ErrConflict", err)
	}

	input := WebhookEvent{
		Provider:   provider,
		ExternalID: "event_external",
		EventType:  "membership.created",
		Payload:    json.RawMessage(`{"membership":"membership_external"}`),
		OccurredAt: time.Now(),
	}
	first, created, err := store.ReceiveWebhookEvent(t.Context(), input)
	if err != nil || !created {
		t.Fatalf("receive first webhook: event=%#v created=%v err=%v", first, created, err)
	}
	repeated, created, err := store.ReceiveWebhookEvent(t.Context(), input)
	if err != nil || created || repeated.ID != first.ID {
		t.Fatalf("receive duplicate webhook: event=%#v created=%v err=%v", repeated, created, err)
	}
	failed, err := store.MarkWebhookEventFailed(
		t.Context(), provider, input.ExternalID, errors.New("temporary failure"),
	)
	if err != nil || failed.Status != WebhookEventFailed || failed.Attempts != 1 {
		t.Fatalf("mark webhook failed: event=%#v err=%v", failed, err)
	}
	processed, err := store.MarkWebhookEventProcessed(t.Context(), provider, input.ExternalID)
	if err != nil || processed.Status != WebhookEventProcessed ||
		processed.Attempts != 2 || processed.ProcessedAt == nil {
		t.Fatalf("mark webhook processed: event=%#v err=%v", processed, err)
	}

	if _, err := store.GetOrganization(
		t.Context(), "00000000-0000-0000-0000-000000000000",
	); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing organization error = %v, want ErrNotFound", err)
	}
}
