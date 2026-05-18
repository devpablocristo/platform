package authz

import "testing"

func TestHasScope(t *testing.T) {
	t.Parallel()

	if !HasScope([]string{"tools:read", "audit:read"}, ScopeAuditRead) {
		t.Fatal("expected scope to be present")
	}
	if HasScope([]string{"tools:read"}, ScopeAdminConsoleWrite) {
		t.Fatal("did not expect missing scope")
	}
}

func TestIsRole(t *testing.T) {
	t.Parallel()

	role := "admin"
	if !IsRole(&role, "viewer", "admin") {
		t.Fatal("expected role match")
	}
}
