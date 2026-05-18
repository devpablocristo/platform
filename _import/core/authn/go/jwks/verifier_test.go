package jwks

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestVerifyTokenEmpty(t *testing.T) {
	t.Parallel()
	v := NewVerifier("http://unused")
	_, err := v.VerifyToken(context.Background(), "")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestVerifyTokenRoundTripRS256(t *testing.T) {
	t.Parallel()

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	pub := &priv.PublicKey
	kid := "unit-kid"

	jwksBody := mustJWKSJSON(t, kid, pub)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(jwksBody)
	}))
	t.Cleanup(srv.Close)

	v := NewVerifier(srv.URL)
	v.cacheTTL = time.Second // acotar caché en test
	v.httpClient = srv.Client()

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"sub": "user-1",
		"iss": "https://issuer.example",
	})
	token.Header["kid"] = kid
	signed, err := token.SignedString(priv)
	if err != nil {
		t.Fatal(err)
	}

	claims, err := v.VerifyToken(context.Background(), signed)
	if err != nil {
		t.Fatal(err)
	}
	if claims["sub"] != "user-1" {
		t.Fatalf("claims: %#v", claims)
	}
}

func TestVerifyTokenWrongKid(t *testing.T) {
	t.Parallel()

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	pub := &priv.PublicKey

	jwksBody := mustJWKSJSON(t, "real-kid", pub)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(jwksBody)
	}))
	t.Cleanup(srv.Close)

	v := NewVerifier(srv.URL)
	v.cacheTTL = time.Second
	v.httpClient = srv.Client()

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{"sub": "x"})
	token.Header["kid"] = "otro-kid"
	signed, err := token.SignedString(priv)
	if err != nil {
		t.Fatal(err)
	}
	_, err = v.VerifyToken(context.Background(), signed)
	if err == nil {
		t.Fatal("expected error for unknown kid")
	}
}

func TestRefreshKeysBadStatus(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	v := NewVerifier(srv.URL)
	v.httpClient = srv.Client()
	_, err := v.VerifyToken(context.Background(), "x.y.z")
	if err == nil {
		t.Fatal("expected error")
	}
}

func mustJWKSJSON(t *testing.T, kid string, pub *rsa.PublicKey) []byte {
	t.Helper()
	nEnc := base64.RawURLEncoding.EncodeToString(pub.N.Bytes())
	eBig := big.NewInt(int64(pub.E))
	eEnc := base64.RawURLEncoding.EncodeToString(eBig.Bytes())
	doc := map[string]any{
		"keys": []map[string]any{
			{
				"kty": "RSA",
				"kid": kid,
				"alg": "RS256",
				"use": "sig",
				"n":   nEnc,
				"e":   eEnc,
			},
		},
	}
	b, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
