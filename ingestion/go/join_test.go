package ingestion

import "testing"

func TestJoinFullText(t *testing.T) {
	t.Parallel()
	arts := []NormalizedArtifact{
		{FullText: "a"},
		{FullText: ""},
		{FullText: "b"},
	}
	if got := JoinFullText(arts); got != "a\n\nb" {
		t.Fatalf("got %q", got)
	}
	if JoinFullText(nil) != "" {
		t.Fatal("expected empty")
	}
}

func TestJoinProvenance(t *testing.T) {
	t.Parallel()
	arts := []NormalizedArtifact{
		{Provenance: Provenance{Engine: "x", EngineVersion: "1"}},
		{Provenance: Provenance{Engine: "y"}},
	}
	if got := JoinProvenance(arts); got != "x:1;y" {
		t.Fatalf("got %q", got)
	}
}
