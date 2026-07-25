package clerk

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestInvitationManagementMapsAndFilters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/organizations/org_123/invitations" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if r.URL.Query().Get("status") != "pending" || r.URL.Query().Get("email_address") != "user@example.com" {
			t.Fatalf("unexpected invitation filters %s", r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`{"data":[{
			"id":"orginv_123",
			"organization_id":"org_123",
			"email_address":"USER@example.com",
			"role":"org:member",
			"status":"pending",
			"url":"https://accounts.example/invitations/orginv_123",
			"private_metadata":{"operation_id":"op_123"},
			"public_metadata":{"source":"admin"},
			"expires_at":1784822400000,
			"created_at":1784818800000,
			"updated_at":1784818800000
		}],"total_count":1}`))
	}))
	defer server.Close()

	client := New(Config{SecretKey: "sk_test", BaseURL: server.URL})
	got, err := client.ListOrgInvitations(context.Background(), "org_123", OrgInvitationListInput{
		Statuses: []string{"pending"},
		Email:    "USER@example.com",
	})
	if err != nil {
		t.Fatalf("ListOrgInvitations: %v", err)
	}
	if len(got) != 1 || got[0].ID != "orginv_123" || got[0].Email != "user@example.com" {
		t.Fatalf("unexpected invitations %+v", got)
	}
	if got[0].ExpiresAt == nil || got[0].Role != "org:member" ||
		got[0].URL != "https://accounts.example/invitations/orginv_123" ||
		got[0].PrivateMetadata["operation_id"] != "op_123" ||
		got[0].PublicMetadata["source"] != "admin" {
		t.Fatalf("unexpected invitation details %+v", got[0])
	}
}

func TestGetAndRevokeInvitation(t *testing.T) {
	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		switch {
		case r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`{"id":"orginv_123","email_address":"user@example.com","role":"org:member","status":"pending"}`))
		case r.Method == http.MethodPost:
			w.WriteHeader(http.StatusNotFound)
		default:
			t.Fatalf("unexpected method %s", r.Method)
		}
	}))
	defer server.Close()

	client := New(Config{SecretKey: "sk_test", BaseURL: server.URL})
	invitation, err := client.GetOrgInvitation(context.Background(), "org_123", "orginv_123")
	if err != nil {
		t.Fatalf("GetOrgInvitation: %v", err)
	}
	if invitation.OrganizationID != "org_123" || invitation.ID != "orginv_123" {
		t.Fatalf("unexpected invitation %+v", invitation)
	}
	if err := client.RevokeOrgInvitation(context.Background(), "org_123", "orginv_123"); err != nil {
		t.Fatalf("RevokeOrgInvitation should tolerate 404: %v", err)
	}
	if len(requests) != 2 {
		t.Fatalf("unexpected requests %+v", requests)
	}
}

func TestMembershipGetAndRevokeAreProviderScoped(t *testing.T) {
	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		switch r.Method {
		case http.MethodGet:
			if r.URL.Query().Get("user_id") != "user_123" {
				t.Fatalf("missing provider user filter %s", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"data":[{
				"id":"orgmem_123",
				"role":"org:member",
				"permissions":["org:members:read"],
				"organization":{"id":"org_123"},
				"public_user_data":{"user_id":"user_123","identifier":"USER@example.com"}
			}],"total_count":1}`))
		case http.MethodDelete:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := New(Config{SecretKey: "sk_test", BaseURL: server.URL})
	membership, ok, err := client.GetOrgMembership(context.Background(), "org_123", "user_123")
	if err != nil || !ok {
		t.Fatalf("GetOrgMembership ok=%v err=%v", ok, err)
	}
	if membership.User.Email != "user@example.com" || len(membership.Permissions) != 1 {
		t.Fatalf("unexpected membership %+v", membership)
	}
	if err := client.RevokeOrgMembership(context.Background(), "org_123", "user_123"); err != nil {
		t.Fatalf("RevokeOrgMembership should tolerate 404: %v", err)
	}
	if requests[0] != "GET /organizations/org_123/memberships" ||
		requests[1] != "DELETE /organizations/org_123/memberships/user_123" {
		t.Fatalf("unexpected requests %+v", requests)
	}
}

func TestSessionManagementMapsAndRevokes(t *testing.T) {
	var revokeHits int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/sessions":
			if r.URL.Query().Get("paginated") != "true" || r.URL.Query().Get("user_id") != "user_123" {
				t.Fatalf("unexpected session query %s", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"data":[{
				"id":"sess_123",
				"user_id":"user_123",
				"client_id":"client_123",
				"status":"active",
				"last_active_organization_id":"org_123",
				"created_at":1784818800000,
				"updated_at":1784818860000,
				"last_active_at":1784818860000,
				"expire_at":1784822400000,
				"abandon_at":1784908800000,
				"latest_activity":{"device_type":"desktop","is_mobile":false,"browser_name":"Firefox"}
			}],"total_count":1}`))
		case r.Method == http.MethodGet && r.URL.Path == "/sessions/sess_123":
			_, _ = w.Write([]byte(`{"id":"sess_123","user_id":"user_123","status":"active"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/sessions/sess_123/revoke":
			revokeHits++
			w.WriteHeader(http.StatusNotFound)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := New(Config{SecretKey: "sk_test", BaseURL: server.URL})
	sessions, err := client.ListSessions(context.Background(), SessionListInput{ProviderUserID: "user_123"})
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	if len(sessions) != 1 || sessions[0].LatestActivity == nil || sessions[0].LatestActivity.BrowserName != "Firefox" {
		t.Fatalf("unexpected sessions %+v", sessions)
	}
	if sessions[0].ExpiresAt.IsZero() || sessions[0].UserID != "user_123" {
		t.Fatalf("unexpected session details %+v", sessions[0])
	}
	session, err := client.GetSession(context.Background(), "sess_123")
	if err != nil || session.ID != "sess_123" {
		t.Fatalf("GetSession: %+v err=%v", session, err)
	}
	if err := client.RevokeSession(context.Background(), "sess_123"); err != nil {
		t.Fatalf("RevokeSession should tolerate 404: %v", err)
	}
	if revokeHits != 1 {
		t.Fatalf("expected one revoke request, got %d", revokeHits)
	}
}

func TestRateLimitErrorExposesRetryAfter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "17")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"errors":[{"message":"rate limited"}]}`))
	}))
	defer server.Close()

	client := New(Config{SecretKey: "sk_test", BaseURL: server.URL})
	_, err := client.GetSession(context.Background(), "sess_123")
	if !IsRateLimited(err) {
		t.Fatalf("expected rate-limited error, got %v", err)
	}
	delay, ok := RetryAfter(err)
	if !ok || delay != 17*time.Second {
		t.Fatalf("unexpected retry delay %s ok=%v", delay, ok)
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Headers.Get("Retry-After") != "17" {
		t.Fatalf("expected response headers on APIError: %#v", err)
	}
}
