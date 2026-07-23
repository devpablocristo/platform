package clerk

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	clerksdk "github.com/clerk/clerk-sdk-go/v2"
	"github.com/go-jose/go-jose/v3"
	"github.com/go-jose/go-jose/v3/jwt"
)

type fixedClock struct {
	now time.Time
}

func (c fixedClock) Now() time.Time {
	return c.now
}

func TestSessionVerifierVerifiesAndNormalizesClaims(t *testing.T) {
	now := time.Date(2026, time.July, 23, 12, 0, 0, 0, time.UTC)
	privateKey := mustRSAKey(t)
	token := mustSessionToken(t, privateKey, "key_1", map[string]any{
		"iss":             "https://example.clerk.accounts.dev",
		"aud":             []string{"pymes-v2-api", "another-audience"},
		"azp":             "http://localhost:15173",
		"sub":             "user_123",
		"sid":             "sess_123",
		"org_id":          "org_123",
		"org_slug":        "cristo-tech",
		"org_role":        "org:admin",
		"org_permissions": []string{"org:members:read"},
		"iat":             now.Add(-time.Minute).Unix(),
		"nbf":             now.Add(-time.Minute).Unix(),
		"exp":             now.Add(time.Minute).Unix(),
	})
	verifier := mustSessionVerifier(t, SessionVerifierConfig{
		PublicKeyPEM:      publicKeyPEM(t, &privateKey.PublicKey),
		Issuer:            "https://example.clerk.accounts.dev",
		Audience:          "pymes-v2-api",
		AuthorizedParties: []string{"http://127.0.0.1:15173", "http://localhost:15173"},
		Clock:             fixedClock{now: now},
	})

	got, err := verifier.VerifySession(context.Background(), token)
	if err != nil {
		t.Fatalf("VerifySession: %v", err)
	}
	if got.Subject != "user_123" || got.SessionID != "sess_123" || got.OrganizationID != "org_123" {
		t.Fatalf("unexpected identity claims: %+v", got)
	}
	if got.OrganizationRole != "org:admin" || len(got.OrganizationPermissions) != 1 {
		t.Fatalf("unexpected organization claims: %+v", got)
	}
	if got.Status != SessionStatusActive || got.AuthorizedParty != "http://localhost:15173" {
		t.Fatalf("unexpected normalized claims: %+v", got)
	}
	if !got.ExpiresAt.Equal(now.Add(time.Minute)) {
		t.Fatalf("unexpected expiry %s", got.ExpiresAt)
	}
}

func TestSessionVerifierFailsClosed(t *testing.T) {
	now := time.Date(2026, time.July, 23, 12, 0, 0, 0, time.UTC)
	privateKey := mustRSAKey(t)
	baseClaims := map[string]any{
		"iss":      "https://example.clerk.accounts.dev",
		"aud":      []string{"pymes-v2-api"},
		"azp":      "http://localhost:15173",
		"sub":      "user_123",
		"sid":      "sess_123",
		"org_id":   "org_123",
		"org_role": "org:member",
		"iat":      now.Add(-time.Minute).Unix(),
		"nbf":      now.Add(-time.Minute).Unix(),
		"exp":      now.Add(time.Minute).Unix(),
	}
	config := SessionVerifierConfig{
		PublicKeyPEM:      publicKeyPEM(t, &privateKey.PublicKey),
		Issuer:            "https://example.clerk.accounts.dev",
		Audience:          "pymes-v2-api",
		AuthorizedParties: []string{"http://localhost:15173"},
		Clock:             fixedClock{now: now},
	}

	tests := []struct {
		name   string
		mutate func(map[string]any)
		target error
	}{
		{
			name: "issuer mismatch",
			mutate: func(claims map[string]any) {
				claims["iss"] = "https://other.clerk.accounts.dev"
			},
			target: ErrInvalidSessionToken,
		},
		{
			name: "audience mismatch",
			mutate: func(claims map[string]any) {
				claims["aud"] = []string{"other-api"}
			},
			target: ErrInvalidSessionToken,
		},
		{
			name: "unauthorized party",
			mutate: func(claims map[string]any) {
				claims["azp"] = "https://evil.example"
			},
			target: ErrInvalidSessionToken,
		},
		{
			name: "expired",
			mutate: func(claims map[string]any) {
				claims["exp"] = now.Add(-time.Second).Unix()
			},
			target: ErrInvalidSessionToken,
		},
		{
			name: "not active yet",
			mutate: func(claims map[string]any) {
				claims["nbf"] = now.Add(time.Minute).Unix()
			},
			target: ErrInvalidSessionToken,
		},
		{
			name: "issued in the future",
			mutate: func(claims map[string]any) {
				claims["iat"] = now.Add(time.Minute).Unix()
			},
			target: ErrInvalidSessionToken,
		},
		{
			name: "missing subject",
			mutate: func(claims map[string]any) {
				delete(claims, "sub")
			},
			target: ErrInvalidSessionToken,
		},
		{
			name: "missing session",
			mutate: func(claims map[string]any) {
				delete(claims, "sid")
			},
			target: ErrInvalidSessionToken,
		},
		{
			name: "missing organization",
			mutate: func(claims map[string]any) {
				delete(claims, "org_id")
			},
			target: ErrOrganizationRequired,
		},
		{
			name: "pending session",
			mutate: func(claims map[string]any) {
				claims["sts"] = "pending"
			},
			target: ErrPendingSession,
		},
		{
			name: "unknown status",
			mutate: func(claims map[string]any) {
				claims["sts"] = "unknown"
			},
			target: ErrInvalidSessionToken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims := cloneClaims(t, baseClaims)
			tt.mutate(claims)
			token := mustSessionToken(t, privateKey, "key_1", claims)
			verifier := mustSessionVerifier(t, config)
			_, err := verifier.VerifySession(context.Background(), token)
			if !errors.Is(err, tt.target) {
				t.Fatalf("expected %v, got %v", tt.target, err)
			}
		})
	}
}

func TestSessionVerifierSupportsIdentityScopeWithoutOrganization(t *testing.T) {
	now := time.Date(2026, time.July, 23, 12, 0, 0, 0, time.UTC)
	privateKey := mustRSAKey(t)
	claims := map[string]any{
		"iss": "https://example.clerk.accounts.dev",
		"aud": []string{"pymes-v2-api"},
		"azp": "http://localhost:15173",
		"sub": "user_123",
		"sid": "sess_123",
		"iat": now.Add(-time.Minute).Unix(),
		"nbf": now.Add(-time.Minute).Unix(),
		"exp": now.Add(time.Minute).Unix(),
	}
	token := mustSessionToken(t, privateKey, "key_1", claims)
	verifier := mustSessionVerifier(t, SessionVerifierConfig{
		PublicKeyPEM:      publicKeyPEM(t, &privateKey.PublicKey),
		Issuer:            "https://example.clerk.accounts.dev",
		Audience:          "pymes-v2-api",
		AuthorizedParties: []string{"http://localhost:15173"},
		Clock:             fixedClock{now: now},
	})

	identity, err := verifier.VerifyIdentity(context.Background(), token)
	if err != nil {
		t.Fatalf("VerifyIdentity() error = %v", err)
	}
	if identity.Subject != "user_123" || identity.SessionID != "sess_123" {
		t.Fatalf("identity claims = %+v", identity)
	}
	if identity.OrganizationID != "" || identity.OrganizationRole != "" {
		t.Fatalf("identity unexpectedly has organization: %+v", identity)
	}

	if _, err := verifier.VerifySession(context.Background(), token); !errors.Is(
		err,
		ErrOrganizationRequired,
	) {
		t.Fatalf("VerifySession() error = %v, want ErrOrganizationRequired", err)
	}
}

func TestIdentityVerificationRejectsPartialOrganizationClaims(t *testing.T) {
	now := time.Date(2026, time.July, 23, 12, 0, 0, 0, time.UTC)
	privateKey := mustRSAKey(t)
	token := mustSessionToken(t, privateKey, "key_1", map[string]any{
		"iss":      "https://example.clerk.accounts.dev",
		"aud":      []string{"pymes-v2-api"},
		"azp":      "http://localhost:15173",
		"sub":      "user_123",
		"sid":      "sess_123",
		"org_role": "org:member",
		"iat":      now.Add(-time.Minute).Unix(),
		"nbf":      now.Add(-time.Minute).Unix(),
		"exp":      now.Add(time.Minute).Unix(),
	})
	verifier := mustSessionVerifier(t, SessionVerifierConfig{
		PublicKeyPEM:      publicKeyPEM(t, &privateKey.PublicKey),
		Issuer:            "https://example.clerk.accounts.dev",
		Audience:          "pymes-v2-api",
		AuthorizedParties: []string{"http://localhost:15173"},
		Clock:             fixedClock{now: now},
	})

	if _, err := verifier.VerifyIdentity(context.Background(), token); !errors.Is(
		err,
		ErrInvalidSessionToken,
	) {
		t.Fatalf("VerifyIdentity() error = %v, want ErrInvalidSessionToken", err)
	}
}

func TestSessionVerifierHonorsClockSkew(t *testing.T) {
	now := time.Date(2026, time.July, 23, 12, 0, 0, 0, time.UTC)
	privateKey := mustRSAKey(t)
	token := mustSessionToken(t, privateKey, "key_1", map[string]any{
		"iss":      "https://example.clerk.accounts.dev",
		"aud":      []string{"pymes-v2-api"},
		"azp":      "http://localhost:15173",
		"sub":      "user_123",
		"sid":      "sess_123",
		"org_id":   "org_123",
		"org_role": "org:member",
		"iat":      now.Add(-time.Minute).Unix(),
		"nbf":      now.Add(-time.Minute).Unix(),
		"exp":      now.Add(-15 * time.Second).Unix(),
	})
	verifier := mustSessionVerifier(t, SessionVerifierConfig{
		PublicKeyPEM:      publicKeyPEM(t, &privateKey.PublicKey),
		Issuer:            "https://example.clerk.accounts.dev",
		Audience:          "pymes-v2-api",
		AuthorizedParties: []string{"http://localhost:15173"},
		ClockSkew:         30 * time.Second,
		Clock:             fixedClock{now: now},
	})
	if _, err := verifier.VerifySession(context.Background(), token); err != nil {
		t.Fatalf("expected token inside clock skew to pass: %v", err)
	}
}

func TestSessionVerifierFetchesAndCachesJWKS(t *testing.T) {
	now := time.Date(2026, time.July, 23, 12, 0, 0, 0, time.UTC)
	privateKey := mustRSAKey(t)
	token := mustSessionToken(t, privateKey, "key_1", map[string]any{
		"iss":      "https://example.clerk.accounts.dev",
		"aud":      []string{"pymes-v2-api"},
		"azp":      "http://localhost:15173",
		"sub":      "user_123",
		"sid":      "sess_123",
		"org_id":   "org_123",
		"org_role": "org:member",
		"iat":      now.Add(-time.Minute).Unix(),
		"nbf":      now.Add(-time.Minute).Unix(),
		"exp":      now.Add(time.Minute).Unix(),
	})

	hits := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if r.URL.Path != "/jwks" {
			t.Fatalf("unexpected JWKS path %q", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer sk_test" {
			t.Fatalf("unexpected authorization header %q", got)
		}
		_, _ = fmt.Fprintf(w, `{"keys":[{"kty":"RSA","kid":"key_1","use":"sig","alg":"RS256","n":%q,"e":"AQAB"}]}`,
			base64.RawURLEncoding.EncodeToString(privateKey.PublicKey.N.Bytes()),
		)
	}))
	defer server.Close()

	verifier := mustSessionVerifier(t, SessionVerifierConfig{
		SecretKey:         "sk_test",
		BaseURL:           server.URL,
		Issuer:            "https://example.clerk.accounts.dev",
		Audience:          "pymes-v2-api",
		AuthorizedParties: []string{"http://localhost:15173"},
		Clock:             fixedClock{now: now},
	})
	for range 2 {
		if _, err := verifier.VerifySession(context.Background(), token); err != nil {
			t.Fatalf("VerifySession: %v", err)
		}
	}
	if hits != 1 {
		t.Fatalf("expected one JWKS fetch, got %d", hits)
	}
}

func TestNewSessionVerifierRequiresStrictConfiguration(t *testing.T) {
	valid := SessionVerifierConfig{
		PublicKeyPEM:      publicKeyPEM(t, &mustRSAKey(t).PublicKey),
		Issuer:            "https://example.clerk.accounts.dev",
		Audience:          "pymes-v2-api",
		AuthorizedParties: []string{"http://localhost:15173"},
	}
	tests := []struct {
		name   string
		mutate func(*SessionVerifierConfig)
	}{
		{"issuer", func(c *SessionVerifierConfig) { c.Issuer = "" }},
		{"audience", func(c *SessionVerifierConfig) { c.Audience = "" }},
		{"authorized parties", func(c *SessionVerifierConfig) { c.AuthorizedParties = nil }},
		{"clock skew", func(c *SessionVerifierConfig) { c.ClockSkew = -time.Second }},
		{"verification key", func(c *SessionVerifierConfig) { c.PublicKeyPEM = "" }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := valid
			tt.mutate(&config)
			if _, err := NewSessionVerifier(config); err == nil {
				t.Fatal("expected configuration error")
			}
		})
	}
}

func mustSessionVerifier(t *testing.T, config SessionVerifierConfig) *SessionVerifier {
	t.Helper()
	verifier, err := NewSessionVerifier(config)
	if err != nil {
		t.Fatalf("NewSessionVerifier: %v", err)
	}
	return verifier
}

func mustRSAKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}
	return key
}

func publicKeyPEM(t *testing.T, key *rsa.PublicKey) string {
	t.Helper()
	raw, err := x509.MarshalPKIXPublicKey(key)
	if err != nil {
		t.Fatalf("marshal public key: %v", err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: raw}))
}

func mustSessionToken(t *testing.T, key *rsa.PrivateKey, keyID string, claims map[string]any) string {
	t.Helper()
	options := (&jose.SignerOptions{}).WithType("JWT").WithHeader("kid", keyID)
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: key}, options)
	if err != nil {
		t.Fatalf("create signer: %v", err)
	}
	token, err := jwt.Signed(signer).Claims(claims).CompactSerialize()
	if err != nil {
		t.Fatalf("serialize token: %v", err)
	}
	return token
}

func cloneClaims(t *testing.T, input map[string]any) map[string]any {
	t.Helper()
	raw, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("marshal claims: %v", err)
	}
	var output map[string]any
	if err := json.Unmarshal(raw, &output); err != nil {
		t.Fatalf("unmarshal claims: %v", err)
	}
	return output
}

var _ clerksdk.Clock = fixedClock{}
