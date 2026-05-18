package clerkwebhook

import (
	"context"
	"encoding/json"
	"testing"

	userdomain "github.com/devpablocristo/core/saas/go/users/usecases/domain"
)

type stubSyncer struct {
	orgID           string
	syncOrgExternal string
	syncOrgName     string
	removeUserID    string
	removeOrgID     string
	removeOrgName   string
}

func (s *stubSyncer) SyncUser(context.Context, string, string, string, *string) (userdomain.User, error) {
	return userdomain.User{}, nil
}

func (s *stubSyncer) SyncOrganization(_ context.Context, orgExternalID, orgName string) (string, error) {
	s.syncOrgExternal = orgExternalID
	s.syncOrgName = orgName
	if s.orgID == "" {
		s.orgID = "org-local-1"
	}
	return s.orgID, nil
}

func (s *stubSyncer) SyncMembership(context.Context, string, string, string, string, *string, string) (userdomain.OrgMember, error) {
	return userdomain.OrgMember{}, nil
}

func (s *stubSyncer) SoftDeleteUser(context.Context, string) error {
	return nil
}

func (s *stubSyncer) RemoveMembership(_ context.Context, userExternalID, orgExternalID, orgName string) error {
	s.removeUserID = userExternalID
	s.removeOrgID = orgExternalID
	s.removeOrgName = orgName
	return nil
}

func TestOnOrganizationCreatedPassesExternalID(t *testing.T) {
	t.Parallel()

	syncer := &stubSyncer{}
	handler := NewHandler(Config{ClerkWebhookSecret: "whsec_test"}, syncer, nil, nil)
	raw, err := json.Marshal(clerkOrganizationData{
		ID:   "org_123",
		Name: "Treasury Ops",
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	if err := handler.onOrganizationCreated(context.Background(), raw); err != nil {
		t.Fatalf("onOrganizationCreated returned error: %v", err)
	}
	if syncer.syncOrgExternal != "org_123" {
		t.Fatalf("expected external id org_123, got %q", syncer.syncOrgExternal)
	}
	if syncer.syncOrgName != "Treasury Ops" {
		t.Fatalf("expected org name Treasury Ops, got %q", syncer.syncOrgName)
	}
}

func TestOnOrganizationMembershipDeletedPrefersExternalID(t *testing.T) {
	t.Parallel()

	syncer := &stubSyncer{}
	handler := NewHandler(Config{ClerkWebhookSecret: "whsec_test"}, syncer, nil, nil)

	payload := clerkMembershipData{}
	payload.Organization.ID = "org_456"
	payload.Organization.Name = "Finance"
	payload.PublicUserData.UserID = "user_789"
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	if err := handler.onOrganizationMembershipDeleted(context.Background(), raw); err != nil {
		t.Fatalf("onOrganizationMembershipDeleted returned error: %v", err)
	}
	if syncer.removeUserID != "user_789" {
		t.Fatalf("expected user id user_789, got %q", syncer.removeUserID)
	}
	if syncer.removeOrgID != "org_456" {
		t.Fatalf("expected org external id org_456, got %q", syncer.removeOrgID)
	}
	if syncer.removeOrgName != "Finance" {
		t.Fatalf("expected org name Finance, got %q", syncer.removeOrgName)
	}
}
