package authn

import "testing"

func TestBearerToken(t *testing.T) {
	t.Parallel()

	token, ok := BearerToken("Bearer abc123")
	if !ok || token != "abc123" {
		t.Fatalf("unexpected bearer parsing: token=%q ok=%v", token, ok)
	}
}

func TestAPIKeyToken(t *testing.T) {
	t.Parallel()

	token, ok := APIKeyToken("", "key-1")
	if !ok || token != "key-1" {
		t.Fatalf("unexpected x-api-key parsing: token=%q ok=%v", token, ok)
	}
	token, ok = APIKeyToken("ApiKey key-2", "")
	if !ok || token != "key-2" {
		t.Fatalf("unexpected auth header parsing: token=%q ok=%v", token, ok)
	}
}
