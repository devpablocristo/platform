package oidc

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

func TestDiscoverAndVerifyToken(t *testing.T) {
	t.Parallel()

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	pub := &priv.PublicKey
	kid := "oidc-kid"
	jwksBytes := mustJWKS(t, kid, pub)

	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		iss := "http://" + r.Host
		doc := DiscoveryDocument{
			Issuer:                iss,
			AuthorizationEndpoint: iss + "/oauth/authorize",
			TokenEndpoint:         iss + "/oauth/token",
			JWKSURI:               iss + "/jwks",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(doc)
	})
	mux.HandleFunc("/jwks", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(jwksBytes)
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	dc := NewDiscoveryClient(srv.URL)
	dc.cacheTTL = time.Second
	dc.httpClient = srv.Client()

	ctx := context.Background()
	doc, err := dc.Discover(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if doc.JWKSURI == "" || doc.Issuer == "" {
		t.Fatalf("doc: %+v", doc)
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"sub": "sub-oidc",
	})
	token.Header["kid"] = kid
	signed, err := token.SignedString(priv)
	if err != nil {
		t.Fatal(err)
	}

	claims, err := dc.VerifyToken(ctx, signed)
	if err != nil {
		t.Fatal(err)
	}
	if claims["sub"] != "sub-oidc" {
		t.Fatalf("claims %#v", claims)
	}
}

func mustJWKS(t *testing.T, kid string, pub *rsa.PublicKey) []byte {
	t.Helper()
	nEnc := base64.RawURLEncoding.EncodeToString(pub.N.Bytes())
	eEnc := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes())
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
