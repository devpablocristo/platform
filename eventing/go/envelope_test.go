package event

import (
	"testing"
	"time"
)

func TestNewAndDecode(t *testing.T) {
	t.Parallel()

	value := New("report.ready", map[string]any{"ok": true}, Metadata{
		TenantID: "acme",
		Source:   "worker",
	}, time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC))

	body, err := Encode(value)
	if err != nil {
		t.Fatalf("Encode returned error: %v", err)
	}
	decoded, err := Decode[map[string]any](body)
	if err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	if decoded.Metadata.Kind != "report.ready" {
		t.Fatalf("unexpected kind: %q", decoded.Metadata.Kind)
	}
	if decoded.Metadata.Version != 1 {
		t.Fatalf("unexpected version: %d", decoded.Metadata.Version)
	}
	if decoded.Metadata.TenantID != "acme" {
		t.Fatalf("unexpected tenant: %q", decoded.Metadata.TenantID)
	}
}
