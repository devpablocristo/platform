package clerk

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateOrganization(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/organizations" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer sk_test" {
			t.Fatalf("unexpected auth header %q", got)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if body["name"] != "Cristo Tech" || body["slug"] != "cristo-tech" {
			t.Fatalf("unexpected body %+v", body)
		}
		_, _ = w.Write([]byte(`{"id":"org_FAKE","name":"Cristo Tech","slug":"cristo-tech"}`))
	}))
	defer server.Close()

	client := New(Config{SecretKey: "sk_test", BaseURL: server.URL})
	got, err := client.CreateOrganization(context.Background(), OrganizationInput{Name: "Cristo Tech", Slug: "cristo-tech"})
	if err != nil {
		t.Fatalf("CreateOrganization: %v", err)
	}
	if got.ID != "org_FAKE" || got.Name != "Cristo Tech" || got.Slug != "cristo-tech" {
		t.Fatalf("unexpected org %+v", got)
	}
}

func TestUpdateOrganizationUsesProviderOrgID(t *testing.T) {
	var path string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		if r.Method != http.MethodPatch {
			t.Fatalf("unexpected method %s", r.Method)
		}
		_, _ = w.Write([]byte(`{"id":"org_PROVIDER","name":"New Name","slug":"new-name"}`))
	}))
	defer server.Close()

	client := New(Config{SecretKey: "sk_test", BaseURL: server.URL})
	got, err := client.UpdateOrganization(context.Background(), "org_PROVIDER", OrganizationInput{Name: "New Name"})
	if err != nil {
		t.Fatalf("UpdateOrganization: %v", err)
	}
	if path != "/organizations/org_PROVIDER" {
		t.Fatalf("expected provider org path, got %q", path)
	}
	if got.ID != "org_PROVIDER" || got.Name != "New Name" {
		t.Fatalf("unexpected org %+v", got)
	}
}

func TestDeleteOrganizationUsesProviderOrgIDAndTolerates404(t *testing.T) {
	var hits []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits = append(hits, r.Method+" "+r.URL.Path)
		if r.Method != http.MethodDelete || r.URL.Path != "/organizations/org_PROVIDER" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := New(Config{SecretKey: "sk_test", BaseURL: server.URL})
	if err := client.DeleteOrganization(context.Background(), "org_PROVIDER"); err != nil {
		t.Fatalf("DeleteOrganization should tolerate 404: %v", err)
	}
	if len(hits) != 1 || hits[0] != "DELETE /organizations/org_PROVIDER" {
		t.Fatalf("unexpected hits %+v", hits)
	}
}

func TestFindUserByEmailSupportsArrayResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users" || r.URL.Query().Get("email_address") != "user@example.com" {
			t.Fatalf("unexpected request %s", r.URL.String())
		}
		_, _ = w.Write([]byte(`[{"id":"user_FAKE","email_addresses":[{"id":"em_1","email_address":"user@example.com"}],"primary_email_address_id":"em_1"}]`))
	}))
	defer server.Close()

	client := New(Config{SecretKey: "sk_test", BaseURL: server.URL})
	got, ok, err := client.FindUserByEmail(context.Background(), "USER@example.com")
	if err != nil {
		t.Fatalf("FindUserByEmail: %v", err)
	}
	if !ok || got.ID != "user_FAKE" || got.Email != "user@example.com" {
		t.Fatalf("unexpected user ok=%v %+v", ok, got)
	}
}

func TestCreateOrgMembershipUsesProviderOrgID(t *testing.T) {
	var seen string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.Method + " " + r.URL.Path
		if !strings.Contains(r.URL.Path, "org_PROVIDER") {
			t.Fatalf("provider org id not used in path %q", r.URL.Path)
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	client := New(Config{SecretKey: "sk_test", BaseURL: server.URL})
	err := client.CreateOrgMembership(context.Background(), OrgMembershipInput{
		ProviderOrgID:  "org_PROVIDER",
		ProviderUserID: "user_123",
		Role:           "org:member",
	})
	if err != nil {
		t.Fatalf("CreateOrgMembership: %v", err)
	}
	if seen != "POST /organizations/org_PROVIDER/memberships" {
		t.Fatalf("unexpected request %q", seen)
	}
}

func TestAPIErrorCarriesStatusAndMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"errors":[{"long_message":"bad request from clerk"}]}`))
	}))
	defer server.Close()

	client := New(Config{SecretKey: "sk_test", BaseURL: server.URL})
	_, err := client.CreateOrganization(context.Background(), OrganizationInput{Name: "bad"})
	if StatusCode(err) != http.StatusUnprocessableEntity {
		t.Fatalf("expected status 422, got %v", err)
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Message("fallback") != "bad request from clerk" {
		t.Fatalf("unexpected api error %#v", err)
	}
}
