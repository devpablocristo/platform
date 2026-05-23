package identityhttp

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	authn "github.com/devpablocristo/platform/authn/go"
	"github.com/devpablocristo/platform/errors/go/domainerr"
)

func TestWithPrincipalSanitizesInboundIdentityHeaders(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/v1/resources", nil)
	req.Header.Set(HeaderOrgID, "spoofed")
	req.Header.Set(HeaderUserID, "spoofed-user")
	req.Header.Set(HeaderAuthScopes, "spoof:scope")

	out := WithPrincipal(req, &authn.Principal{
		OrgID:  "argos-local-org",
		Actor:  "argos",
		Role:   "service",
		Scopes: []string{"nexus:findings:read", "nexus:findings:write"},
		Claims: map[string]any{"service_principal": true},
	}, "api_key")

	ctx := FromRequest(out)
	if ctx.OrgID != "argos-local-org" || ctx.Actor != "argos" || ctx.AuthMethod != "api_key" {
		t.Fatalf("unexpected identity context: %+v", ctx)
	}
	if !ctx.ServicePrincipal {
		t.Fatalf("expected service principal")
	}
	if got := out.Header.Get(HeaderOrgID); got != "argos-local-org" {
		t.Fatalf("expected sanitized org header, got %q", got)
	}
	if !HasAnyScope(out, "nexus:findings:write") {
		t.Fatalf("expected findings write scope")
	}
}

func TestFromRequestParsesHeaderFallback(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/v1/resources", nil)
	req.Header.Set(HeaderOrgID, "org-a")
	req.Header.Set(HeaderUserID, "user-a")
	req.Header.Set(HeaderAuthScopes, "a,b;c+d")
	req.Header.Set(HeaderAuthMethod, "jwt")
	req.Header.Set(HeaderServicePrincipal, "yes")

	ctx := FromRequest(req)
	if ctx.OrgID != "org-a" || ctx.Actor != "user-a" || ctx.AuthMethod != "jwt" {
		t.Fatalf("unexpected identity context: %+v", ctx)
	}
	if !ctx.ServicePrincipal {
		t.Fatalf("expected service principal")
	}
	for _, scope := range []string{"a", "b", "c", "d"} {
		if !HasScope(req, scope) {
			t.Fatalf("expected scope %q in %+v", scope, ctx.Scopes)
		}
	}
}

func TestEffectiveOrgID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/v1/resources", nil)
	req = WithPrincipal(req, &authn.Principal{OrgID: "org-a", Scopes: []string{"read"}}, "api_key")

	if got, ok := EffectiveOrgID(req, "", "cross"); !ok || got != "org-a" {
		t.Fatalf("expected principal org, got %q ok=%v", got, ok)
	}
	if _, ok := EffectiveOrgID(req, "org-b", "cross"); ok {
		t.Fatalf("expected cross-org mismatch to fail")
	}

	cross := httptest.NewRequest(http.MethodGet, "/v1/resources", nil)
	cross = WithPrincipal(cross, &authn.Principal{OrgID: "org-a", Scopes: []string{"cross"}}, "api_key")
	if got, ok := EffectiveOrgID(cross, "org-b", "cross"); !ok || got != "org-b" {
		t.Fatalf("expected requested cross org, got %q ok=%v", got, ok)
	}
}

func TestRequireOrgMatch_MatchesPrincipal(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/v1/resources", nil)
	req = WithPrincipal(req, &authn.Principal{OrgID: "org-a", Scopes: []string{"read"}}, "api_key")

	got, err := RequireOrgMatch(req, "", "cross")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "org-a" {
		t.Errorf("got %q, want org-a", got)
	}
}

func TestRequireOrgMatch_MismatchReturnsDomainErr(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/v1/resources", nil)
	req = WithPrincipal(req, &authn.Principal{OrgID: "org-a", Scopes: []string{"read"}}, "api_key")

	_, err := RequireOrgMatch(req, "org-b", "cross")
	if err == nil {
		t.Fatal("expected mismatch error")
	}
	if !domainerr.IsForbidden(err) {
		t.Errorf("expected FORBIDDEN kind, got %v", err)
	}
	if !errors.Is(err, domainerr.TenantMismatch()) {
		t.Errorf("expected errors.Is TenantMismatch, got %v", err)
	}
}

func TestRequireOrgMatch_CrossOrgBypass(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/v1/resources", nil)
	req = WithPrincipal(req, &authn.Principal{OrgID: "org-a", Scopes: []string{"cross"}}, "api_key")

	got, err := RequireOrgMatch(req, "org-b", "cross")
	if err != nil {
		t.Fatalf("unexpected error with cross_org scope: %v", err)
	}
	if got != "org-b" {
		t.Errorf("got %q, want org-b (cross_org should pass requested)", got)
	}
}

func TestRequireOrgMatch_PrincipalWithoutOrgFails(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/v1/resources", nil)
	req = WithPrincipal(req, &authn.Principal{OrgID: "", Scopes: []string{"read"}}, "api_key")

	_, err := RequireOrgMatch(req, "", "cross")
	if err == nil {
		t.Fatal("expected error when principal has no org")
	}
	if !errors.Is(err, domainerr.TenantMissing()) {
		t.Errorf("expected TenantMissing, got %v", err)
	}
}

func TestRequireOrgMatch_NoAuthContextAllows(t *testing.T) {
	// Dev mode: sin auth context → pasa el requested
	req := httptest.NewRequest(http.MethodGet, "/v1/resources", nil)
	got, err := RequireOrgMatch(req, "anything", "cross")
	if err != nil {
		t.Fatalf("dev-mode (no auth) should not fail: %v", err)
	}
	if got != "anything" {
		t.Errorf("got %q, want anything (passthrough)", got)
	}
}
