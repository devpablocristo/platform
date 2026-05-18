package governanceclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSubmitRequest(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/requests" {
			t.Errorf("expected /v1/requests, got %s", r.URL.Path)
		}
		if r.Header.Get("X-API-Key") != "test-key" {
			t.Errorf("expected X-API-Key=test-key, got %s", r.Header.Get("X-API-Key"))
		}
		if r.Header.Get("Idempotency-Key") != "idem-123" {
			t.Errorf("expected Idempotency-Key=idem-123, got %s", r.Header.Get("Idempotency-Key"))
		}

		var body SubmitRequestBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body.ActionType != "appointment.cancel" {
			t.Errorf("expected action_type=appointment.cancel, got %s", body.ActionType)
		}

		w.WriteHeader(http.StatusCreated)
		resp := SubmitResponse{
			RequestID:      "req-abc",
			Decision:       "allow",
			RiskLevel:      "low",
			DecisionReason: "policy matched",
			Status:         "allowed",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "test-key")
	resp, err := c.SubmitRequest(context.Background(), "idem-123", SubmitRequestBody{
		RequesterType: "service",
		RequesterID:   "pymes-ai",
		ActionType:    "appointment.cancel",
		TargetSystem:  "pymes",
	})
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	if resp.RequestID != "req-abc" {
		t.Errorf("expected request_id=req-abc, got %s", resp.RequestID)
	}
	if resp.Decision != "allow" {
		t.Errorf("expected decision=allow, got %s", resp.Decision)
	}
}

func TestSimulateRequest(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/requests/simulate" {
			t.Errorf("expected /v1/requests/simulate, got %s", r.URL.Path)
		}
		if r.Header.Get("X-API-Key") != "test-key" {
			t.Errorf("expected X-API-Key=test-key, got %s", r.Header.Get("X-API-Key"))
		}

		var body SimulateRequestBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body.ActionType != "procurement.submit" {
			t.Errorf("expected action_type=procurement.submit, got %s", body.ActionType)
		}

		w.WriteHeader(http.StatusOK)
		matched := "auto-approve-low-spend"
		json.NewEncoder(w).Encode(SimulateResponse{
			Decision:             DecisionAllow,
			RiskLevel:            "low",
			DecisionReason:       "matched policy auto-approve-low-spend",
			Status:               StatusAllowed,
			PolicyMatched:        &matched,
			RiskAssessment:       json.RawMessage(`{"factors":[]}`),
			WouldRequireApproval: false,
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "test-key")
	resp, err := c.SimulateRequest(context.Background(), SimulateRequestBody{
		RequesterType: "service",
		RequesterID:   "pymes-procurement",
		ActionType:    "procurement.submit",
		TargetSystem:  "pymes",
		Params:        map[string]any{"amount": 500, "currency": "USD"},
	})
	if err != nil {
		t.Fatalf("simulate: %v", err)
	}
	if resp.Decision != DecisionAllow {
		t.Errorf("expected decision=%s, got %s", DecisionAllow, resp.Decision)
	}
	if resp.Status != StatusAllowed {
		t.Errorf("expected status=%s, got %s", StatusAllowed, resp.Status)
	}
	if resp.WouldRequireApproval {
		t.Error("expected would_require_approval=false")
	}
	if resp.PolicyMatched == nil || *resp.PolicyMatched != "auto-approve-low-spend" {
		t.Errorf("expected policy_matched=auto-approve-low-spend, got %v", resp.PolicyMatched)
	}
}

func TestSimulateRequestRequireApproval(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(SimulateResponse{
			Decision:             DecisionRequireApproval,
			RiskLevel:            "high",
			DecisionReason:       "amount exceeds threshold",
			Status:               StatusPendingApproval,
			WouldRequireApproval: true,
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "test-key")
	resp, err := c.SimulateRequest(context.Background(), SimulateRequestBody{
		RequesterType: "service",
		RequesterID:   "pymes-procurement",
		ActionType:    "procurement.submit",
		Params:        map[string]any{"amount": 75000},
	})
	if err != nil {
		t.Fatalf("simulate: %v", err)
	}
	if resp.Decision != DecisionRequireApproval {
		t.Errorf("expected decision=%s, got %s", DecisionRequireApproval, resp.Decision)
	}
	if !resp.WouldRequireApproval {
		t.Error("expected would_require_approval=true")
	}
}

func TestSimulateRequestServerError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"code":"INTERNAL","message":"upstream policy store down"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "test-key")
	_, err := c.SimulateRequest(context.Background(), SimulateRequestBody{
		RequesterType: "service",
		RequesterID:   "x",
		ActionType:    "procurement.submit",
	})
	if err == nil {
		t.Fatal("expected error on 500, got nil")
	}
}

func TestGetRequest(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/requests/req-123" {
			t.Errorf("expected /v1/requests/req-123, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(RequestSummary{
			ID:       "req-123",
			Decision: "allow",
			Status:   "allowed",
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "test-key")
	summary, st, err := c.GetRequest(context.Background(), "req-123")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if st != http.StatusOK {
		t.Errorf("expected 200, got %d", st)
	}
	if summary.Decision != "allow" {
		t.Errorf("expected decision=allow, got %s", summary.Decision)
	}
}

func TestGetRequestNotFound(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "test-key")
	_, st, err := c.GetRequest(context.Background(), "no-exist")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if st != http.StatusNotFound {
		t.Errorf("expected 404, got %d", st)
	}
}

func TestReportResult(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/requests/req-456/result" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "test-key")
	err := c.ReportResult(context.Background(), "req-456", true, 150, "ok")
	if err != nil {
		t.Fatalf("report: %v", err)
	}
}

func TestListPolicies(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/policies" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"policies":[]}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "test-key")
	st, raw, err := c.ListPolicies(context.Background())
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if st != http.StatusOK {
		t.Errorf("expected 200, got %d", st)
	}
	if len(raw) == 0 {
		t.Error("expected non-empty response")
	}
}

func TestWithTenantIDScopesRequest(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Org-ID"); got != "tenant-123" {
			t.Errorf("expected Nexus tenant scope header tenant-123, got %q", got)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data":[]}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "test-key")
	st, _, err := c.ListPolicies(context.Background(), WithTenantID("tenant-123"))
	if err != nil {
		t.Fatalf("list policies: %v", err)
	}
	if st != http.StatusOK {
		t.Fatalf("expected 200, got %d", st)
	}
}

func TestParseErrorBody(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"valid json", `{"code":"NOT_FOUND","message":"request not found"}`, "request not found"},
		{"no message", `{"code":"INTERNAL"}`, `{"code":"INTERNAL"}`},
		{"plain text", `server error`, "server error"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ParseErrorBody([]byte(tt.input))
			if got != tt.want {
				t.Errorf("ParseErrorBody(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
