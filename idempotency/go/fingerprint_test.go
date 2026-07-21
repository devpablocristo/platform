package idempotency

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestFingerprintRequestIsStableAndRestoresBody(t *testing.T) {
	request, err := http.NewRequest(http.MethodPost, "https://example.test/orders?dry_run=true", strings.NewReader(`{"total":"10.50"}`))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "Application/JSON")

	first, err := FingerprintRequest(request, 1024)
	if err != nil {
		t.Fatal(err)
	}
	second, err := FingerprintRequest(request, 1024)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("fingerprint changed: %q != %q", first, second)
	}
	body, err := io.ReadAll(request.Body)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(body), `{"total":"10.50"}`; got != want {
		t.Fatalf("body was not restored: got %q want %q", got, want)
	}
}

func TestFingerprintRequestSeparatesTargetsAndBodies(t *testing.T) {
	requestA := newRequest(t, http.MethodPost, "/commands?mode=fast", "alpha")
	requestB := newRequest(t, http.MethodPost, "/commands?mode=slow", "alpha")
	requestC := newRequest(t, http.MethodPost, "/commands?mode=fast", "beta")

	fingerprintA, err := FingerprintRequest(requestA, 1024)
	if err != nil {
		t.Fatal(err)
	}
	fingerprintB, err := FingerprintRequest(requestB, 1024)
	if err != nil {
		t.Fatal(err)
	}
	fingerprintC, err := FingerprintRequest(requestC, 1024)
	if err != nil {
		t.Fatal(err)
	}
	if fingerprintA == fingerprintB || fingerprintA == fingerprintC {
		t.Fatal("different request targets or bodies produced the same fingerprint")
	}
}

func TestFingerprintRequestRejectsOversizedBody(t *testing.T) {
	request := newRequest(t, http.MethodPost, "/commands", "12345")
	_, err := FingerprintRequest(request, 4)
	if err != ErrRequestTooLarge {
		t.Fatalf("expected ErrRequestTooLarge, got %v", err)
	}
}

func newRequest(t *testing.T, method, target, body string) *http.Request {
	t.Helper()
	request, err := http.NewRequest(method, "https://example.test"+target, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	return request
}
