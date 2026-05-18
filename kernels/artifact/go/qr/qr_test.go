package qr

import (
	"bytes"
	"testing"
)

func TestPNGProducesPNG(t *testing.T) {
	t.Parallel()

	body, err := PNG("https://example.com/pay/123", 128)
	if err != nil {
		t.Fatalf("PNG returned error: %v", err)
	}
	if !bytes.HasPrefix(body, []byte{0x89, 'P', 'N', 'G'}) {
		t.Fatalf("expected png header, got %v", body[:4])
	}
}
