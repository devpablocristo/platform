package clerk

import "testing"

func TestClaimsAdapterMapsStandardClaims(t *testing.T) {
	claims := map[string]any{
		"sub":             "user_123",
		"email":           "owner@example.com",
		"org_id":          "org_abc",
		"org_role":        "org:admin",
		"org_permissions": []any{"orgs:read", "users:write"},
	}

	principal := ClaimsAdapter{}.Principal(claims)

	if principal.OrgID != "org_abc" || principal.TenantID != "org_abc" {
		t.Fatalf("unexpected org mapping: %#v", principal)
	}
	if principal.Actor != "owner@example.com" || principal.Role != "admin" {
		t.Fatalf("unexpected actor/role: %#v", principal)
	}
	if principal.AuthMethod != Provider {
		t.Fatalf("unexpected auth method: %q", principal.AuthMethod)
	}
	if len(principal.Scopes) != 2 || principal.Scopes[0] != "orgs:read" {
		t.Fatalf("unexpected scopes: %#v", principal.Scopes)
	}
}

func TestClaimsAdapterMapsAbbreviatedOrgClaims(t *testing.T) {
	claims := map[string]any{
		"sub":   "user_123",
		"o":     map[string]any{"id": "org_nested", "rol": "org:member", "per": []any{"members:read"}},
		"email": "member@example.com",
	}

	identity := ClaimsAdapter{}.ExternalIdentity(claims)

	if identity.OrgID != "org_nested" || identity.ExternalOrgID != "org_nested" {
		t.Fatalf("unexpected external identity: %#v", identity)
	}
	if identity.Role != "member" {
		t.Fatalf("unexpected role %q", identity.Role)
	}
	if len(identity.Scopes) != 1 || identity.Scopes[0] != "members:read" {
		t.Fatalf("unexpected scopes: %#v", identity.Scopes)
	}
}
