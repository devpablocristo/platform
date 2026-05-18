package pdf

import (
	"bytes"
	"testing"
)

func TestSimpleProducesPDF(t *testing.T) {
	t.Parallel()

	body, err := Simple(Document{
		Title:    "Report",
		Subtitle: "Reusable export",
		Lines: []Line{
			{Label: "Tenant", Value: "acme"},
			{Label: "Status", Value: "ready"},
		},
	})
	if err != nil {
		t.Fatalf("Simple returned error: %v", err)
	}
	if !bytes.HasPrefix(body, []byte("%PDF")) {
		t.Fatalf("expected PDF signature, got %q", string(body[:4]))
	}
}
