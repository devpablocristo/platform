package identity_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/devpablocristo/platform/errors/go/domainerr"
	"github.com/devpablocristo/platform/kernels/saas/go/identity"
)

type stubVerifier struct {
	claims map[string]any
	err    error
}

func (s stubVerifier) VerifyToken(context.Context, string) (map[string]any, error) {
	return s.claims, s.err
}

type stubMemberships map[string][]identity.ActorTenant

func (m stubMemberships) TenantsForActor(_ context.Context, sub string) ([]identity.ActorTenant, error) {
	return m[sub], nil
}

func firebaseClaims() map[string]any {
	return map[string]any{"iss": "https://securetoken.google.com/proj", "aud": "proj", "sub": "uid-1"}
}

func newResolver(mem stubMemberships) *identity.MembershipResolver {
	return identity.NewMembershipResolver(
		stubVerifier{claims: firebaseClaims()},
		mem,
		identity.MembershipConfig{Issuer: "https://securetoken.google.com/proj", Audience: "proj"},
	)
}

func TestMembershipResolver_Policy_0_1_N(t *testing.T) {
	ctx := context.Background()

	t.Run("0 memberships -> forbidden", func(t *testing.T) {
		_, err := newResolver(stubMemberships{}).Verify(ctx, "tok")
		if !domainerr.IsKind(err, domainerr.KindForbidden) {
			t.Fatalf("want forbidden, got %v", err)
		}
	})

	t.Run("1 membership -> usa esa", func(t *testing.T) {
		mem := stubMemberships{"uid-1": {{TenantID: "t-A", Role: "viewer", Scopes: []string{"api.read"}}}}
		p, err := newResolver(mem).Verify(ctx, "tok")
		if err != nil {
			t.Fatal(err)
		}
		if p.OrgID != "t-A" || p.TenantID != "t-A" || p.Role != "viewer" || p.Actor != "uid-1" {
			t.Fatalf("Principal inesperado: %+v", p)
		}
		if strings.Join(p.Scopes, ",") != "api.read" || p.AuthMethod != "jwt" {
			t.Fatalf("scopes/authMethod inesperados: %+v", p)
		}
	})

	t.Run(">1 sin requested -> selection required (validation)", func(t *testing.T) {
		mem := stubMemberships{"uid-1": {{TenantID: "t-A"}, {TenantID: "t-B"}}}
		_, err := newResolver(mem).Verify(ctx, "tok")
		if !domainerr.IsKind(err, domainerr.KindValidation) {
			t.Fatalf("want validation, got %v", err)
		}
	})

	t.Run(">1 con requested -> usa el pedido", func(t *testing.T) {
		mem := stubMemberships{"uid-1": {{TenantID: "t-A", Role: "viewer"}, {TenantID: "t-B", Role: "manager"}}}
		rctx := identity.WithRequestedTenant(ctx, "t-B")
		p, err := newResolver(mem).Verify(rctx, "tok")
		if err != nil {
			t.Fatal(err)
		}
		if p.OrgID != "t-B" || p.Role != "manager" {
			t.Fatalf("Principal inesperado: %+v", p)
		}
	})

	t.Run("requested que no matchea -> forbidden (nunca cae a otro)", func(t *testing.T) {
		mem := stubMemberships{"uid-1": {{TenantID: "t-A"}}}
		rctx := identity.WithRequestedTenant(ctx, "t-Z")
		_, err := newResolver(mem).Verify(rctx, "tok")
		if !domainerr.IsKind(err, domainerr.KindForbidden) {
			t.Fatalf("want forbidden, got %v", err)
		}
	})
}

func TestMembershipResolver_RejectsBadIssuerAudience(t *testing.T) {
	mem := stubMemberships{"uid-1": {{TenantID: "t-A"}}}

	t.Run("issuer inválido", func(t *testing.T) {
		r := identity.NewMembershipResolver(
			stubVerifier{claims: map[string]any{"iss": "https://evil/", "aud": "proj", "sub": "uid-1"}},
			mem, identity.MembershipConfig{Issuer: "https://securetoken.google.com/proj", Audience: "proj"})
		if _, err := r.Verify(context.Background(), "tok"); !domainerr.IsKind(err, domainerr.KindUnauthorized) {
			t.Fatalf("want unauthorized, got %v", err)
		}
	})

	t.Run("audience inválido", func(t *testing.T) {
		r := identity.NewMembershipResolver(
			stubVerifier{claims: map[string]any{"iss": "https://securetoken.google.com/proj", "aud": "otro", "sub": "uid-1"}},
			mem, identity.MembershipConfig{Issuer: "https://securetoken.google.com/proj", Audience: "proj"})
		if _, err := r.Verify(context.Background(), "tok"); !domainerr.IsKind(err, domainerr.KindUnauthorized) {
			t.Fatalf("want unauthorized, got %v", err)
		}
	})
}

func TestMembershipResolver_MissingSubjectAndVerifierError(t *testing.T) {
	cfg := identity.MembershipConfig{Issuer: "https://securetoken.google.com/proj", Audience: "proj"}

	t.Run("sin sub -> unauthorized", func(t *testing.T) {
		r := identity.NewMembershipResolver(
			stubVerifier{claims: map[string]any{"iss": "https://securetoken.google.com/proj", "aud": "proj"}},
			stubMemberships{}, cfg)
		if _, err := r.Verify(context.Background(), "tok"); !domainerr.IsKind(err, domainerr.KindUnauthorized) {
			t.Fatalf("want unauthorized, got %v", err)
		}
	})

	t.Run("verifier falla -> unauthorized", func(t *testing.T) {
		r := identity.NewMembershipResolver(
			stubVerifier{err: errors.New("bad token")},
			stubMemberships{}, cfg)
		if _, err := r.Verify(context.Background(), "tok"); !domainerr.IsKind(err, domainerr.KindUnauthorized) {
			t.Fatalf("want unauthorized, got %v", err)
		}
	})
}
