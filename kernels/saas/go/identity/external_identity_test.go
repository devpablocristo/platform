package identity

import "testing"

func TestExternalIdentityFromClaimsMapsConfiguredOrgPrincipal(t *testing.T) {
	claims := map[string]any{
		"sub":          "user_123",
		"mail":         "owner@example.com",
		"organization": "org_abc",
		"role_name":    "org:admin",
		"permissions":  []any{"orgs:read", "users:write"},
	}

	identity := ExternalIdentityFromClaims(claims, ExternalIdentityMapping{
		Provider:    "external-idp",
		ActorClaim:  "sub",
		EmailClaim:  "mail",
		OrgClaim:    "organization",
		RoleClaim:   "role_name",
		ScopesClaim: "permissions",
	})
	principal := PrincipalFromExternalIdentity(identity, ExternalIdentityMapping{})

	if principal.OrgID != "org_abc" || principal.TenantID != "org_abc" {
		t.Fatalf("unexpected org mapping: %#v", principal)
	}
	if principal.Actor != "owner@example.com" || principal.Role != "admin" {
		t.Fatalf("unexpected actor/role: %#v", principal)
	}
	if principal.AuthMethod != "external-idp" {
		t.Fatalf("unexpected auth method: %#v", principal.AuthMethod)
	}
	if len(principal.Scopes) != 2 || principal.Scopes[0] != "orgs:read" {
		t.Fatalf("unexpected scopes: %#v", principal.Scopes)
	}
}

func TestExternalIdentityFromClaimsAllowsUserScopedPrincipalWithoutOrg(t *testing.T) {
	claims := map[string]any{
		"sub":    "user-1",
		"email":  "person@example.com",
		"groups": "admin,caregiver",
		"scope":  "records:read records:write",
	}

	identity := ExternalIdentityFromClaims(claims, ExternalIdentityMapping{
		Provider:    "user-pool",
		GroupsClaim: "groups",
	})
	principal := PrincipalFromExternalIdentity(identity, ExternalIdentityMapping{})

	if principal.OrgID != "" || principal.TenantID != "" {
		t.Fatalf("user-scoped mapping must not invent org ids: %#v", principal)
	}
	if principal.Actor != "person@example.com" || principal.Role != "admin" {
		t.Fatalf("unexpected actor/role: %#v", principal)
	}
	if len(principal.Scopes) != 2 {
		t.Fatalf("unexpected scopes: %#v", principal.Scopes)
	}
}
