package internaljwt_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	authn "github.com/devpablocristo/platform/authn/go"
	"github.com/devpablocristo/platform/authn/go/internaljwt"
)

const testSecret = "test-internal-jwt-secret-aaaaaaaaaaaaaaaaaaaa"

func sign(t *testing.T, claims map[string]any, secret string) string {
	t.Helper()
	header := map[string]any{"alg": "HS256", "typ": "JWT"}
	hJSON, _ := json.Marshal(header)
	cJSON, _ := json.Marshal(claims)
	h := base64.RawURLEncoding.EncodeToString(hJSON)
	c := base64.RawURLEncoding.EncodeToString(cJSON)
	unsigned := h + "." + c
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(unsigned))
	s := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return unsigned + "." + s
}

func TestNewAuthenticator_NilOnEmptySecret(t *testing.T) {
	t.Parallel()
	if got := internaljwt.NewAuthenticator(internaljwt.Config{Secret: ""}); got != nil {
		t.Error("expected nil authenticator on empty secret")
	}
	if got := internaljwt.NewAuthenticator(internaljwt.Config{Secret: "   "}); got != nil {
		t.Error("expected nil authenticator on whitespace secret")
	}
}

func TestAuthenticate_HappyPath(t *testing.T) {
	t.Parallel()
	a := internaljwt.NewAuthenticator(internaljwt.Config{
		Secret: testSecret, Issuer: "axis-bff", Audience: "companion",
	})
	if a == nil {
		t.Fatal("nil authenticator")
	}
	token := sign(t, map[string]any{
		"iss":     "axis-bff",
		"aud":     "companion",
		"sub":     "user-1",
		"org_id":  "org-a",
		"role":    "admin",
		"scope":   "a b c",
		"exp":     float64(time.Now().Add(time.Minute).Unix()),
	}, testSecret)
	p, err := a.Authenticate(context.Background(), authn.BearerCredential{Token: token})
	if err != nil {
		t.Fatalf("authenticate: %v", err)
	}
	if p.OrgID != "org-a" {
		t.Errorf("OrgID=%q", p.OrgID)
	}
	if p.Actor != "user-1" {
		t.Errorf("Actor=%q", p.Actor)
	}
	if p.Role != "admin" {
		t.Errorf("Role=%q", p.Role)
	}
	if len(p.Scopes) != 3 {
		t.Errorf("Scopes=%v", p.Scopes)
	}
	if p.AuthMethod != internaljwt.AuthMethod {
		t.Errorf("AuthMethod=%q", p.AuthMethod)
	}
}

func TestAuthenticate_RejectsWrongSecret(t *testing.T) {
	t.Parallel()
	a := internaljwt.NewAuthenticator(internaljwt.Config{Secret: testSecret})
	token := sign(t, map[string]any{"sub": "x", "iss": ""}, "OTHER_SECRET_xxxxxxxxxxxxxxxxxxxxxxxx")
	if _, err := a.Authenticate(context.Background(), authn.BearerCredential{Token: token}); err == nil {
		t.Error("expected signature error")
	}
}

func TestAuthenticate_RejectsWrongIssuer(t *testing.T) {
	t.Parallel()
	a := internaljwt.NewAuthenticator(internaljwt.Config{Secret: testSecret, Issuer: "axis-bff"})
	token := sign(t, map[string]any{"sub": "x", "iss": "other-iss"}, testSecret)
	_, err := a.Authenticate(context.Background(), authn.BearerCredential{Token: token})
	if err == nil || !strings.Contains(err.Error(), "issuer") {
		t.Errorf("expected issuer error, got %v", err)
	}
}

func TestAuthenticate_RejectsWrongAudience(t *testing.T) {
	t.Parallel()
	a := internaljwt.NewAuthenticator(internaljwt.Config{Secret: testSecret, Audience: "companion"})
	token := sign(t, map[string]any{"sub": "x", "aud": "ponti"}, testSecret)
	_, err := a.Authenticate(context.Background(), authn.BearerCredential{Token: token})
	if err == nil || !strings.Contains(err.Error(), "audience") {
		t.Errorf("expected audience error, got %v", err)
	}
}

func TestAuthenticate_AcceptsAudienceFromAzp(t *testing.T) {
	t.Parallel()
	a := internaljwt.NewAuthenticator(internaljwt.Config{Secret: testSecret, Audience: "companion"})
	token := sign(t, map[string]any{"sub": "x", "aud": "ponti", "azp": "companion"}, testSecret)
	if _, err := a.Authenticate(context.Background(), authn.BearerCredential{Token: token}); err != nil {
		t.Errorf("azp should be accepted: %v", err)
	}
}

func TestAuthenticate_RejectsExpired(t *testing.T) {
	t.Parallel()
	a := internaljwt.NewAuthenticator(internaljwt.Config{Secret: testSecret})
	token := sign(t, map[string]any{
		"sub": "x",
		"exp": float64(time.Now().Add(-time.Minute).Unix()),
	}, testSecret)
	_, err := a.Authenticate(context.Background(), authn.BearerCredential{Token: token})
	if err == nil || !strings.Contains(err.Error(), "expired") {
		t.Errorf("expected expired error, got %v", err)
	}
}

func TestAuthenticate_RejectsNotYetActive(t *testing.T) {
	t.Parallel()
	a := internaljwt.NewAuthenticator(internaljwt.Config{Secret: testSecret})
	token := sign(t, map[string]any{
		"sub": "x",
		"nbf": float64(time.Now().Add(time.Minute).Unix()),
	}, testSecret)
	_, err := a.Authenticate(context.Background(), authn.BearerCredential{Token: token})
	if err == nil || !strings.Contains(err.Error(), "not active") {
		t.Errorf("expected nbf error, got %v", err)
	}
}

func TestAuthenticate_RejectsMissingSub(t *testing.T) {
	t.Parallel()
	a := internaljwt.NewAuthenticator(internaljwt.Config{Secret: testSecret})
	token := sign(t, map[string]any{"org_id": "x"}, testSecret)
	_, err := a.Authenticate(context.Background(), authn.BearerCredential{Token: token})
	if err == nil || !strings.Contains(err.Error(), "subject") {
		t.Errorf("expected missing-subject error, got %v", err)
	}
}

func TestAuthenticate_AcceptsTenantIdFallback(t *testing.T) {
	t.Parallel()
	a := internaljwt.NewAuthenticator(internaljwt.Config{Secret: testSecret})
	token := sign(t, map[string]any{
		"sub":       "x",
		"tenant_id": "tnt-1", // fallback en lugar de org_id
	}, testSecret)
	p, err := a.Authenticate(context.Background(), authn.BearerCredential{Token: token})
	if err != nil {
		t.Fatalf("authenticate: %v", err)
	}
	if p.OrgID != "tnt-1" {
		t.Errorf("expected tenant_id fallback, got %q", p.OrgID)
	}
}

func TestAuthenticate_AcceptsScpAsArray(t *testing.T) {
	t.Parallel()
	a := internaljwt.NewAuthenticator(internaljwt.Config{Secret: testSecret})
	token := sign(t, map[string]any{
		"sub": "x",
		"scp": []any{"a", "b", "c"},
	}, testSecret)
	p, err := a.Authenticate(context.Background(), authn.BearerCredential{Token: token})
	if err != nil {
		t.Fatalf("authenticate: %v", err)
	}
	if len(p.Scopes) != 3 {
		t.Errorf("expected 3 scopes, got %v", p.Scopes)
	}
}

func TestNewFallback_FirstSucceeds(t *testing.T) {
	t.Parallel()
	good := internaljwt.NewAuthenticator(internaljwt.Config{Secret: testSecret})
	bad := internaljwt.NewAuthenticator(internaljwt.Config{Secret: "OTHER_SECRET_yyyyyyyyyyyyyyyyy"})
	fb := internaljwt.NewFallback(good, bad)
	token := sign(t, map[string]any{"sub": "x"}, testSecret)
	if _, err := fb.Authenticate(context.Background(), authn.BearerCredential{Token: token}); err != nil {
		t.Errorf("expected success via good authenticator: %v", err)
	}
}

func TestNewFallback_SecondSucceeds(t *testing.T) {
	t.Parallel()
	bad := internaljwt.NewAuthenticator(internaljwt.Config{Secret: "BAD_SECRET_zzzzzzzzzzzzzzzzzzzzz"})
	good := internaljwt.NewAuthenticator(internaljwt.Config{Secret: testSecret})
	fb := internaljwt.NewFallback(bad, good)
	token := sign(t, map[string]any{"sub": "x"}, testSecret)
	if _, err := fb.Authenticate(context.Background(), authn.BearerCredential{Token: token}); err != nil {
		t.Errorf("expected success via fallback: %v", err)
	}
}

func TestNewFallback_AllNilReturnsNil(t *testing.T) {
	t.Parallel()
	if fb := internaljwt.NewFallback(nil, nil); fb != nil {
		t.Error("expected nil from all-nil fallback")
	}
}

func TestVerifier_RejectsNonHS256(t *testing.T) {
	t.Parallel()
	a := internaljwt.NewAuthenticator(internaljwt.Config{Secret: testSecret})
	// Token with alg=none
	header := `{"alg":"none","typ":"JWT"}`
	claims := `{"sub":"x"}`
	h := base64.RawURLEncoding.EncodeToString([]byte(header))
	c := base64.RawURLEncoding.EncodeToString([]byte(claims))
	token := h + "." + c + "."
	_, err := a.Authenticate(context.Background(), authn.BearerCredential{Token: token})
	if err == nil || !strings.Contains(err.Error(), "unsupported alg") {
		t.Errorf("expected unsupported-alg, got %v", err)
	}
}
