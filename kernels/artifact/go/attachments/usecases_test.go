package attachments

import (
	"strings"
	"testing"
)

func TestBuildStorageKey(t *testing.T) {
	t.Parallel()

	key, err := BuildStorageKey("acme", "Invoice", "inv_1", "../factura.pdf")
	if err != nil {
		t.Fatalf("BuildStorageKey returned error: %v", err)
	}
	if !strings.Contains(key, "acme/invoice/inv_1/") {
		t.Fatalf("unexpected key: %q", key)
	}
	if strings.Contains(key, "..") {
		t.Fatalf("key should be sanitized: %q", key)
	}
}

func TestBuildDownloadLink(t *testing.T) {
	t.Parallel()

	link, err := BuildDownloadLink("https://api.example.com/v1", "att_1", 0)
	if err != nil {
		t.Fatalf("BuildDownloadLink returned error: %v", err)
	}
	if !strings.Contains(link.URL, "/attachments/att_1/download") {
		t.Fatalf("unexpected URL: %q", link.URL)
	}
}
