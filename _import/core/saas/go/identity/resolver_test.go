package identity

import (
	"context"
	"errors"
	"testing"

	"github.com/devpablocristo/core/errors/go/domainerr"
)

func TestClaimsResolverVerify(t *testing.T) {
	t.Parallel()

	resolver := NewClaimsResolver(staticTokenVerifier{
		claims: map[string]any{
			"iss":       "issuer-1",
			"aud":       "aud-1",
			"tenant_id": "acme",
			"sub":       "user-1",
			"role":      "admin",
			"scopes":    "chat:run billing:read",
		},
	}, ClaimsConfig{
		Issuer:   "issuer-1",
		Audience: "aud-1",
	})

	principal, err := resolver.Verify(context.Background(), "token-1")
	if err != nil {
		t.Fatalf("Verify returned error: %v", err)
	}
	if principal.TenantID != "acme" {
		t.Fatalf("unexpected tenant id: %q", principal.TenantID)
	}
	if principal.Actor != "user-1" {
		t.Fatalf("unexpected actor: %q", principal.Actor)
	}
	if len(principal.Scopes) != 2 {
		t.Fatalf("unexpected scopes: %#v", principal.Scopes)
	}
}

type stubOrgResolver struct {
	tenantID string
	ok       bool
	err      error
	got      string
}

func (s *stubOrgResolver) FindOrgIDByExternalID(_ context.Context, externalID string) (string, bool, error) {
	s.got = externalID
	return s.tenantID, s.ok, s.err
}

func TestResolvePrincipalAcceptsCompactClaimsAndExternalOrgLookup(t *testing.T) {
	t.Parallel()

	resolver := &stubOrgResolver{tenantID: "tenant-internal-1", ok: true}
	uc := NewUsecasesWithOrgResolver(staticTokenVerifier{claims: map[string]any{
		"iss": "https://issuer.example",
		"aud": "example-service",
		"sub": "user_abc",
		"o": map[string]any{
			"id":  "org_2s9x",
			"rol": "org:admin",
			"per": []string{"admin:console:read", "billing:read"},
		},
	}}, resolver, Config{
		Issuer:   "https://issuer.example",
		Audience: "example-service",
	})

	principal, err := uc.ResolvePrincipal(context.Background(), "token")
	if err != nil {
		t.Fatalf("ResolvePrincipal returned error: %v", err)
	}
	if resolver.got != "org_2s9x" {
		t.Fatalf("expected external org lookup, got %q", resolver.got)
	}
	if principal.TenantID != "tenant-internal-1" {
		t.Fatalf("expected tenant-internal-1, got %q", principal.TenantID)
	}
	if principal.Role != "admin" {
		t.Fatalf("expected role admin, got %q", principal.Role)
	}
	if len(principal.Scopes) != 2 || principal.Scopes[0] != "admin:console:read" {
		t.Fatalf("unexpected scopes: %#v", principal.Scopes)
	}
}

func TestResolvePrincipalAcceptsIssuerWithTrailingSlashMismatch(t *testing.T) {
	t.Parallel()

	uc := NewUsecasesWithOrgResolver(staticTokenVerifier{claims: map[string]any{
		"iss":       "https://issuer.example/",
		"tenant_id": "acme",
		"sub":       "user-1",
	}}, nil, Config{
		Issuer: "https://issuer.example",
	})

	principal, err := uc.ResolvePrincipal(context.Background(), "token")
	if err != nil {
		t.Fatalf("ResolvePrincipal returned error: %v", err)
	}
	if principal.TenantID != "acme" {
		t.Fatalf("unexpected tenant id: %q", principal.TenantID)
	}
}

func TestResolvePrincipalRejectsUnknownExternalOrg(t *testing.T) {
	t.Parallel()

	uc := NewUsecasesWithOrgResolver(staticTokenVerifier{claims: map[string]any{
		"org_id": "org_missing",
	}}, &stubOrgResolver{}, Config{OrgClaim: "org_id"})

	_, err := uc.ResolvePrincipal(context.Background(), "token")
	if err == nil {
		t.Fatal("expected error for unknown org")
	}
	var de domainerr.Error
	if !errors.As(err, &de) {
		t.Fatalf("expected domainerr.Error, got %T", err)
	}
	if de.Kind() != domainerr.KindUnauthorized {
		t.Fatalf("expected KindUnauthorized, got %s", de.Kind())
	}
}

type staticTokenVerifier struct {
	claims map[string]any
}

func (s staticTokenVerifier) VerifyToken(context.Context, string) (map[string]any, error) {
	return s.claims, nil
}
