package pagination

import (
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"
)

func TestNormalizeLimit(t *testing.T) {
	t.Parallel()

	config := Config{DefaultLimit: 25, MaxLimit: 50}
	if got := NormalizeLimit(0, config); got != 25 {
		t.Fatalf("unexpected default limit: %d", got)
	}
	if got := NormalizeLimit(999, config); got != 50 {
		t.Fatalf("unexpected max clamp: %d", got)
	}
	if got := NormalizeLimit(10, config); got != 10 {
		t.Fatalf("unexpected explicit limit: %d", got)
	}
}

func TestParseParams(t *testing.T) {
	t.Parallel()

	params, err := ParseParams("30", " cursor-1 ", DefaultConfig())
	if err != nil {
		t.Fatalf("ParseParams returned error: %v", err)
	}
	if params.Limit != 30 {
		t.Fatalf("unexpected limit: %d", params.Limit)
	}
	if params.Cursor != "cursor-1" {
		t.Fatalf("unexpected cursor: %q", params.Cursor)
	}
}

func TestParseParamsRejectsInvalidLimit(t *testing.T) {
	t.Parallel()

	if _, err := ParseParams("-1", "", DefaultConfig()); err == nil {
		t.Fatal("expected error for invalid limit")
	}
}

func TestBuildResultClonesItems(t *testing.T) {
	t.Parallel()

	items := []string{"a", "b"}
	result := BuildResult(items, true, "next-1")
	items[0] = "changed"

	if result.Items[0] != "a" {
		t.Fatalf("expected cloned items, got %#v", result.Items)
	}
	if !result.HasMore {
		t.Fatal("expected has_more to be true")
	}
	if result.NextCursor != "next-1" {
		t.Fatalf("unexpected next cursor: %q", result.NextCursor)
	}
}

func TestTimeIDCursorRoundTrip(t *testing.T) {
	t.Parallel()

	createdAt := time.Date(2026, 7, 8, 20, 26, 31, 123, time.FixedZone("ART", -3*60*60))
	encoded, err := EncodeTimeIDCursor(TimeIDCursor{
		CreatedAt: createdAt,
		ID:        "  123e4567-e89b-12d3-a456-426614174000  ",
	})
	if err != nil {
		t.Fatalf("EncodeTimeIDCursor returned error: %v", err)
	}

	decoded, ok, err := DecodeTimeIDCursor(encoded)
	if err != nil {
		t.Fatalf("DecodeTimeIDCursor returned error: %v", err)
	}
	if !ok {
		t.Fatal("expected cursor to be present")
	}
	if decoded.ID != "123e4567-e89b-12d3-a456-426614174000" {
		t.Fatalf("unexpected cursor id: %q", decoded.ID)
	}
	if !decoded.CreatedAt.Equal(createdAt.UTC()) {
		t.Fatalf("unexpected cursor created_at: %s", decoded.CreatedAt)
	}
}

func TestDecodeTimeIDCursorEmpty(t *testing.T) {
	t.Parallel()

	decoded, ok, err := DecodeTimeIDCursor("  ")
	if err != nil {
		t.Fatalf("DecodeTimeIDCursor returned error: %v", err)
	}
	if ok {
		t.Fatal("expected empty cursor to be absent")
	}
	if !decoded.CreatedAt.IsZero() || decoded.ID != "" {
		t.Fatalf("expected zero cursor, got %#v", decoded)
	}
}

func TestDecodeTimeIDCursorRejectsInvalidCursor(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"invalid base64": "not-base64!",
		"invalid json":   base64.RawURLEncoding.EncodeToString([]byte("{")),
		"missing id": mustEncodeCursorPayload(t, map[string]any{
			"created_at": time.Date(2026, 7, 8, 20, 26, 31, 0, time.UTC),
		}),
		"missing created_at": mustEncodeCursorPayload(t, map[string]any{
			"id": "123e4567-e89b-12d3-a456-426614174000",
		}),
	}

	for name, raw := range cases {
		name, raw := name, raw
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, ok, err := DecodeTimeIDCursor(raw); err == nil || ok {
				t.Fatalf("expected invalid cursor error and ok=false, got ok=%v err=%v", ok, err)
			}
		})
	}
}

func mustEncodeCursorPayload(t *testing.T, payload map[string]any) string {
	t.Helper()

	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("json.Marshal returned error: %v", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw)
}
