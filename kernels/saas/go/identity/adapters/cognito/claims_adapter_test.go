package cognito

import "testing"

func TestClaimsAdapterMapsUserPoolPrincipalWithoutOrg(t *testing.T) {
	claims := map[string]any{
		"sub":            "user-123",
		"email":          "person@example.com",
		"cognito:groups": []any{"admin", "caregiver"},
		"scope":          "records:read records:write",
	}

	principal := ClaimsAdapter{}.Principal(claims)

	if principal.OrgID != "" || principal.TenantID != "" {
		t.Fatalf("user pool mapping must not invent org ids: %#v", principal)
	}
	if principal.Actor != "person@example.com" || principal.Role != "admin" {
		t.Fatalf("unexpected actor/role: %#v", principal)
	}
	if principal.AuthMethod != Provider {
		t.Fatalf("unexpected auth method: %q", principal.AuthMethod)
	}
	if len(principal.Scopes) != 2 {
		t.Fatalf("unexpected scopes: %#v", principal.Scopes)
	}
}

func TestClaimsAdapterMapsOptionalOrgClaim(t *testing.T) {
	claims := map[string]any{
		"sub":    "user-123",
		"email":  "member@example.com",
		"org_id": "org_abc",
		"role":   "org:member",
		"scp":    []any{"members:read"},
	}

	identity := ClaimsAdapter{}.ExternalIdentity(claims)

	if identity.OrgID != "org_abc" || identity.ExternalOrgID != "org_abc" {
		t.Fatalf("unexpected org mapping: %#v", identity)
	}
	if identity.Role != "member" {
		t.Fatalf("unexpected role %q", identity.Role)
	}
	if len(identity.Scopes) != 1 || identity.Scopes[0] != "members:read" {
		t.Fatalf("unexpected scopes: %#v", identity.Scopes)
	}
}
