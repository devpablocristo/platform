package session_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/devpablocristo/core/saas/go/identity"
	kerneldomain "github.com/devpablocristo/core/saas/go/kernel/usecases/domain"
	saasmiddleware "github.com/devpablocristo/core/saas/go/middleware"
	"github.com/devpablocristo/core/saas/go/session"
)

type stubJWTVerifier struct {
	p kerneldomain.Principal
}

func (s stubJWTVerifier) Verify(ctx context.Context, token string) (kerneldomain.Principal, error) {
	_ = ctx
	_ = token
	return s.p, nil
}

func TestHandleSession_OK(t *testing.T) {
	t.Parallel()
	want := kerneldomain.Principal{
		TenantID:   "org-1",
		Actor:      "user_1",
		Role:       "viewer",
		Scopes:     []string{"a"},
		AuthMethod: "jwt",
	}
	mux := http.NewServeMux()
	session.RegisterProtected(mux, saasmiddleware.NewAuthMiddleware(
		stubJWTVerifier{p: want},
		nil,
	))
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	req, err := http.NewRequest(http.MethodGet, srv.URL+"/session", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer test-token")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = res.Body.Close() })
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d", res.StatusCode)
	}
	var got kerneldomain.Principal
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.TenantID != want.TenantID || got.Actor != want.Actor || got.Role != want.Role {
		t.Fatalf("got %+v want %+v", got, want)
	}
}

func TestHandleSession_Unauthorized(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	session.RegisterProtected(mux, saasmiddleware.NewAuthMiddleware(nil, nil))
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	req, err := http.NewRequest(http.MethodGet, srv.URL+"/session", nil)
	if err != nil {
		t.Fatal(err)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = res.Body.Close() })
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status %d", res.StatusCode)
	}
}

var _ identity.PrincipalVerifier = stubJWTVerifier{}
