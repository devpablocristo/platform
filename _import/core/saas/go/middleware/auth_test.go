package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/devpablocristo/core/security/go/contextkeys"
	kerneldomain "github.com/devpablocristo/core/saas/go/kernel/usecases/domain"
	"github.com/google/uuid"
)

func TestAuthMiddlewareWithBearer(t *testing.T) {
	t.Parallel()

	tenantID := uuid.NewString()
	handler := AuthMiddleware(fakeVerifier{
		principal: kerneldomain.Principal{TenantID: tenantID, Actor: "user-1", Role: "admin", Scopes: []string{"admin:console:read"}, AuthMethod: "bearer"},
	}, nil, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal, ok := PrincipalFromContext(r.Context())
		if !ok {
			t.Fatal("expected principal in context")
		}
		if principal.TenantID != tenantID {
			t.Fatalf("unexpected principal: %#v", principal)
		}
		if got, _ := r.Context().Value(ctxkeys.Actor).(string); got != "user-1" {
			t.Fatalf("expected actor in ctxkeys, got %q", got)
		}
		if got, _ := r.Context().Value(ctxkeys.Role).(string); got != "admin" {
			t.Fatalf("expected role in ctxkeys, got %q", got)
		}
		if got, _ := r.Context().Value(ctxkeys.AuthMethod).(string); got != "bearer" {
			t.Fatalf("expected auth method in ctxkeys, got %q", got)
		}
		orgID, ok := r.Context().Value(ctxkeys.OrgID).(uuid.UUID)
		if !ok || orgID.String() != tenantID {
			t.Fatalf("expected parsed org id in ctxkeys, got %#v ok=%v", orgID, ok)
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
	req.Header.Set("Authorization", "Bearer token-1")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
}

func TestAuthMiddlewareWithAPIKey(t *testing.T) {
	t.Parallel()

	handler := AuthMiddleware(nil, fakeVerifier{
		principal: kerneldomain.Principal{TenantID: "acme", Actor: "key-1", AuthMethod: "api_key"},
	}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal, ok := PrincipalFromContext(r.Context())
		if !ok || principal.Actor != "key-1" {
			t.Fatalf("unexpected principal: %#v ok=%v", principal, ok)
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
	req.Header.Set("X-API-Key", "secret")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
}

func TestAuthMiddlewareRejectsMissingCredentials(t *testing.T) {
	t.Parallel()

	handler := AuthMiddleware(nil, nil, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/me", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("expected json error payload: %v", err)
	}
}

func TestNewAuthMiddlewareWrapsHandler(t *testing.T) {
	t.Parallel()

	wrapped := NewAuthMiddleware(fakeVerifier{
		principal: kerneldomain.Principal{TenantID: "tenant-1", AuthMethod: "jwt"},
	}, nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
	req.Header.Set("Authorization", "Bearer token")

	wrapped(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := PrincipalFromContext(r.Context()); !ok {
			t.Fatal("expected principal in context")
		}
		w.WriteHeader(http.StatusNoContent)
	})).ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
}

type fakeVerifier struct {
	principal kerneldomain.Principal
	err       error
}

func (f fakeVerifier) Verify(context.Context, string) (kerneldomain.Principal, error) {
	return f.principal, f.err
}
