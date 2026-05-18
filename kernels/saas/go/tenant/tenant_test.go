package tenant

import "testing"

func TestNormalizeSlug(t *testing.T) {
	t.Parallel()

	if got := NormalizeSlug(" ACME / Prod Team "); got != "acme-prod-team" {
		t.Fatalf("unexpected slug: %q", got)
	}
}

func TestNormalizeRole(t *testing.T) {
	t.Parallel()

	if got := NormalizeRole("ADMIN"); got != "admin" {
		t.Fatalf("unexpected role: %q", got)
	}
	if got := NormalizeRole("unknown"); got != "viewer" {
		t.Fatalf("unexpected fallback role: %q", got)
	}
}
