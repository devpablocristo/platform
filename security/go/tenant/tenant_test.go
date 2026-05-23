package tenant_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/devpablocristo/platform/errors/go/domainerr"
	ctxkeys "github.com/devpablocristo/platform/security/go/contextkeys"
	"github.com/devpablocristo/platform/security/go/tenant"
)

func TestID_IsZero(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   tenant.ID
		want bool
	}{
		{"", true},
		{"   ", true},
		{"\t\n", true},
		{"a", false},
		{"local-dev-org", false},
	}
	for _, c := range cases {
		if got := c.in.IsZero(); got != c.want {
			t.Errorf("ID(%q).IsZero() = %v, want %v", string(c.in), got, c.want)
		}
	}
}

func TestID_UUID_RoundTrip(t *testing.T) {
	t.Parallel()
	u := uuid.New()
	id := tenant.FromUUID(u)
	if id.IsZero() {
		t.Fatal("FromUUID produced empty ID")
	}
	got, err := id.UUID()
	if err != nil {
		t.Fatalf("UUID() returned error: %v", err)
	}
	if got != u {
		t.Errorf("roundtrip mismatch: got %v, want %v", got, u)
	}
}

func TestID_UUID_RejectsNonUUID(t *testing.T) {
	t.Parallel()
	id := tenant.FromString("local-dev-org")
	if _, err := id.UUID(); err == nil {
		t.Error("expected error parsing non-UUID, got nil")
	}
}

func TestID_UUID_RejectsEmpty(t *testing.T) {
	t.Parallel()
	var id tenant.ID
	if _, err := id.UUID(); err == nil {
		t.Error("expected error on empty ID, got nil")
	}
}

func TestFromUUID_Nil(t *testing.T) {
	t.Parallel()
	if id := tenant.FromUUID(uuid.Nil); !id.IsZero() {
		t.Errorf("FromUUID(Nil) = %q, want empty", string(id))
	}
}

func TestFromString_Trims(t *testing.T) {
	t.Parallel()
	if id := tenant.FromString("  org-1  "); id.String() != "org-1" {
		t.Errorf("FromString trimmed = %q, want %q", string(id), "org-1")
	}
}

func TestWithID_WritesBothKeys(t *testing.T) {
	t.Parallel()
	ctx := tenant.WithID(context.Background(), tenant.FromString("acme"))
	if got, _ := ctx.Value(ctxkeys.OrgID).(string); got != "acme" {
		t.Errorf("OrgID key = %q, want %q", got, "acme")
	}
	if got, _ := ctx.Value(ctxkeys.TenantID).(string); got != "acme" {
		t.Errorf("TenantID key = %q, want %q", got, "acme")
	}
}

func TestWithID_EmptyDoesNotMutate(t *testing.T) {
	t.Parallel()
	base := context.Background()
	got := tenant.WithID(base, "")
	if got.Value(ctxkeys.OrgID) != nil {
		t.Error("expected no OrgID written for empty id")
	}
	if got.Value(ctxkeys.TenantID) != nil {
		t.Error("expected no TenantID written for empty id")
	}
}

func TestWithID_NilContextSafe(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("WithID(nil, ...) panicked: %v", r)
		}
	}()
	ctx := tenant.WithID(nil, tenant.FromString("a")) //nolint:staticcheck // intentional
	if _, ok := tenant.FromContext(ctx); !ok {
		t.Error("expected tenant readable after WithID(nil, ...)")
	}
}

func TestFromContext_PrefersTenantIDOverOrgID(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	ctx = context.WithValue(ctx, ctxkeys.OrgID, "from-org")
	ctx = context.WithValue(ctx, ctxkeys.TenantID, "from-tenant")
	got, ok := tenant.FromContext(ctx)
	if !ok {
		t.Fatal("expected found")
	}
	if got.String() != "from-tenant" {
		t.Errorf("got %q, want from-tenant (TenantID precedence)", got)
	}
}

func TestFromContext_FallsBackToOrgID(t *testing.T) {
	t.Parallel()
	ctx := context.WithValue(context.Background(), ctxkeys.OrgID, "only-org")
	got, ok := tenant.FromContext(ctx)
	if !ok {
		t.Fatal("expected found via OrgID")
	}
	if got.String() != "only-org" {
		t.Errorf("got %q, want only-org", got)
	}
}

func TestFromContext_Missing(t *testing.T) {
	t.Parallel()
	if _, ok := tenant.FromContext(context.Background()); ok {
		t.Error("expected not found on empty ctx")
	}
}

func TestFromContext_NilContext(t *testing.T) {
	t.Parallel()
	if _, ok := tenant.FromContext(nil); ok { //nolint:staticcheck // intentional
		t.Error("expected not found on nil ctx")
	}
}

func TestFromContext_CoercesUUIDValue(t *testing.T) {
	t.Parallel()
	u := uuid.New()
	ctx := context.WithValue(context.Background(), ctxkeys.TenantID, u)
	got, ok := tenant.FromContext(ctx)
	if !ok {
		t.Fatal("expected found")
	}
	if got.String() != u.String() {
		t.Errorf("got %q, want %q", got, u.String())
	}
}

func TestRequire_FailsClosedOnMissing(t *testing.T) {
	t.Parallel()
	_, err := tenant.Require(context.Background())
	if err == nil {
		t.Fatal("expected error on missing tenant")
	}
	if !domainerr.IsForbidden(err) {
		t.Errorf("expected FORBIDDEN kind, got %v", err)
	}
	if !errors.Is(err, domainerr.TenantMissing()) {
		t.Errorf("expected errors.Is TenantMissing, got %v", err)
	}
}

func TestRequire_ReturnsID(t *testing.T) {
	t.Parallel()
	ctx := tenant.WithID(context.Background(), "acme-corp")
	id, err := tenant.Require(ctx)
	if err != nil {
		t.Fatalf("Require returned error: %v", err)
	}
	if id.String() != "acme-corp" {
		t.Errorf("got %q, want acme-corp", id)
	}
}

func TestRequireUUID_HappyPath(t *testing.T) {
	t.Parallel()
	u := uuid.New()
	ctx := tenant.WithID(context.Background(), tenant.FromUUID(u))
	got, err := tenant.RequireUUID(ctx)
	if err != nil {
		t.Fatalf("RequireUUID returned error: %v", err)
	}
	if got != u {
		t.Errorf("got %v, want %v", got, u)
	}
}

func TestRequireUUID_FailsOnNonUUIDID(t *testing.T) {
	t.Parallel()
	ctx := tenant.WithID(context.Background(), "not-a-uuid")
	_, err := tenant.RequireUUID(ctx)
	if err == nil {
		t.Fatal("expected error on non-UUID tenant")
	}
	if !domainerr.IsValidation(err) {
		t.Errorf("expected VALIDATION_ERROR, got %v", err)
	}
}

func TestRequireUUID_FailsOnMissing(t *testing.T) {
	t.Parallel()
	_, err := tenant.RequireUUID(context.Background())
	if err == nil {
		t.Fatal("expected error on missing tenant")
	}
	if !domainerr.IsForbidden(err) {
		t.Errorf("expected FORBIDDEN kind on missing tenant, got %v", err)
	}
}

func TestStrictMode_SettableInTests(t *testing.T) {
	tenant.SetStrictMode(true)
	if !tenant.StrictModeEnabled() {
		t.Error("expected strict mode enabled after SetStrictMode(true)")
	}
	tenant.SetStrictMode(false)
	if tenant.StrictModeEnabled() {
		t.Error("expected strict mode disabled after SetStrictMode(false)")
	}
}
