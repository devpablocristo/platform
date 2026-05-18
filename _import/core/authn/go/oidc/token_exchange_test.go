package oidc

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestGeneratePKCE(t *testing.T) {
	t.Parallel()
	p, err := GeneratePKCE()
	if err != nil {
		t.Fatal(err)
	}
	if len(p.CodeVerifier) < 40 || len(p.CodeChallenge) < 40 {
		t.Fatalf("short pkce: %+v", p)
	}
}

func TestGenerateStateAndNonce(t *testing.T) {
	t.Parallel()
	s, err := GenerateState()
	if err != nil || len(s) < 10 {
		t.Fatalf("state: %q err=%v", s, err)
	}
	n, err := GenerateNonce()
	if err != nil || len(n) < 10 {
		t.Fatalf("nonce: %q err=%v", n, err)
	}
}

func TestAuthorizationURLContainsParams(t *testing.T) {
	t.Parallel()

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

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	d := NewDiscoveryClient(srv.URL)
	d.httpClient = srv.Client()
	d.cacheTTL = time.Millisecond

	pkce, err := GeneratePKCE()
	if err != nil {
		t.Fatal(err)
	}
	ex := NewTokenExchanger(d, "client-id", "", "https://app/callback")
	urlStr, err := ex.AuthorizationURL(t.Context(), "st1", "no1", pkce, []string{"openid", "email"})
	if err != nil {
		t.Fatal(err)
	}
	for _, part := range []string{"response_type=code", "client_id=client-id", "state=st1", "nonce=no1", "code_challenge="} {
		if !strings.Contains(urlStr, part) {
			t.Fatalf("missing %q in %s", part, urlStr)
		}
	}
}
