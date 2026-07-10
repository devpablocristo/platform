package firebase_test

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/devpablocristo/platform/authn/go/jwks"
	"github.com/devpablocristo/platform/kernels/saas/go/identity"
	"github.com/devpablocristo/platform/kernels/saas/go/identity/adapters/firebase"
)

const (
	testProject = "ponti-preview"
	testKID     = "fb-key-1"
	testUID     = "firebase-uid-abc123"
)

func b64(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }

func signRS256(t *testing.T, key *rsa.PrivateKey, claims map[string]any) string {
	t.Helper()
	header, _ := json.Marshal(map[string]any{"alg": "RS256", "typ": "JWT", "kid": testKID})
	payload, _ := json.Marshal(claims)
	signingInput := b64(header) + "." + b64(payload)
	sum := sha256.Sum256([]byte(signingInput))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, sum[:])
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return signingInput + "." + b64(sig)
}

func jwkServer(t *testing.T, pub *rsa.PublicKey) *httptest.Server {
	t.Helper()
	doc, _ := json.Marshal(map[string]any{"keys": []map[string]any{{
		"kid": testKID, "kty": "RSA", "alg": "RS256", "use": "sig",
		"n": b64(pub.N.Bytes()), "e": b64(big.NewInt(int64(pub.E)).Bytes()),
	}}})
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(doc)
	}))
}

func idToken(t *testing.T, key *rsa.PrivateKey, extra map[string]any) string {
	now := time.Now()
	claims := map[string]any{
		"iss": firebase.IssuerFor(testProject), "aud": testProject,
		"sub": testUID, "user_id": testUID, "email": "user@ponti.local",
		"iat": now.Unix(), "exp": now.Add(time.Hour).Unix(),
	}
	for k, v := range extra {
		claims[k] = v
	}
	return signRS256(t, key, claims)
}

func TestFirebase_HelpersAndConfig(t *testing.T) {
	if got := firebase.IssuerFor("proj-1"); got != "https://securetoken.google.com/proj-1" {
		t.Fatalf("IssuerFor = %q", got)
	}
	cfg := firebase.ClaimsConfig(testProject)
	if cfg.Issuer != "https://securetoken.google.com/ponti-preview" || cfg.Audience != testProject {
		t.Fatalf("ClaimsConfig issuer/aud inesperados: %+v", cfg)
	}
	if cfg.OrgClaim != "org_id" || cfg.ActorClaim != "sub" {
		t.Fatalf("ClaimsConfig claims inesperados: %+v", cfg)
	}
}

func TestFirebase_ClaimsAdapter_WithCustomClaims(t *testing.T) {
	claims := map[string]any{
		"sub": testUID, "email": "u@ponti.local",
		"org_id": "org-9", "role": "manager", "scopes": "api.read api.write",
	}
	p := firebase.ClaimsAdapter{}.Principal(claims)
	if p.OrgID != "org-9" || p.Role != "manager" || p.AuthMethod != firebase.Provider {
		t.Fatalf("Principal inesperado: %+v", p)
	}
	// Actor cae a Email cuando está presente (PrincipalFromExternalIdentity prioriza email).
	if p.Actor != "u@ponti.local" {
		t.Fatalf("Actor = %q, want email", p.Actor)
	}
	if strings.Join(p.Scopes, ",") != "api.read,api.write" {
		t.Fatalf("Scopes = %v", p.Scopes)
	}
}

// Integración real: verifier JWKS de firebase + ClaimsConfig + ClaimsResolver del kernel.
func TestFirebase_Integration_VerifierAndResolver(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	srv := jwkServer(t, &key.PublicKey)
	defer srv.Close()

	// firebase.NewVerifier apunta al JWKS de securetoken en prod; acá lo sustituimos por el server local
	// (mismo formato JWK n/e) para probar el path completo sin red.
	verifier := jwks.NewVerifier(srv.URL)
	resolver := identity.NewClaimsResolver(verifier, firebase.ClaimsConfig(testProject))

	token := idToken(t, key, map[string]any{"org_id": "org-9", "role": "viewer", "scopes": "api.read"})
	p, err := resolver.ResolvePrincipal(context.Background(), token)
	if err != nil {
		t.Fatalf("ResolvePrincipal: %v", err)
	}
	if p.OrgID != "org-9" || p.Actor != testUID || p.Role != "viewer" {
		t.Fatalf("Principal inesperado: %+v", p)
	}

	// Sanity: el JWKS de prod de firebase es el endpoint n/e de securetoken.
	if firebase.JWKSURL != "https://www.googleapis.com/service_accounts/v1/jwk/securetoken@system.gserviceaccount.com" {
		t.Fatalf("JWKSURL inesperado: %s", firebase.JWKSURL)
	}
}
