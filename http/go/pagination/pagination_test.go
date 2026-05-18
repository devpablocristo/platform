package pagination

import "testing"

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
