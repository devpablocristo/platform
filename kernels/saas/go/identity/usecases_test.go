package identity

import (
	"context"
	"testing"
)

func TestLegacyUsecasesResolvePrincipalAcceptsInternalTenantClaim(t *testing.T) {
	t.Parallel()

	uc := NewUsecases(staticTokenVerifier{claims: map[string]any{
		"iss":      "https://issuer.example",
		"aud":      []string{"example-service"},
		"org_id":   "tenant-123",
		"sub":      "user_123",
		"role":     "admin",
		"scopes":   []any{"admin:console:read", "billing:read"},
		"org_role": "org:secops",
	}}, Config{
		Issuer:      "https://issuer.example",
		Audience:    "example-service",
		OrgClaim:    "org_id",
		RoleClaim:   "role",
		ScopesClaim: "scopes",
		ActorClaim:  "sub",
	})

	principal, err := uc.ResolvePrincipal(context.Background(), "token")
	if err != nil {
		t.Fatalf("ResolvePrincipal returned error: %v", err)
	}
	if principal.TenantID != "tenant-123" {
		t.Fatalf("expected tenant-123, got %q", principal.TenantID)
	}
	if principal.Actor != "user_123" {
		t.Fatalf("expected actor user_123, got %q", principal.Actor)
	}
	if principal.Role != "admin" {
		t.Fatalf("expected role admin, got %q", principal.Role)
	}
	if len(principal.Scopes) != 2 {
		t.Fatalf("expected 2 scopes, got %#v", principal.Scopes)
	}
}
