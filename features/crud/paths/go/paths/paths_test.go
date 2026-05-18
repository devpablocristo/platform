package paths

import "testing"

func TestSegmentsNonEmpty(t *testing.T) {
	for _, s := range []string{SegmentArchived, SegmentArchive, SegmentRestore, SegmentHard} {
		if s == "" {
			t.Fatal("empty segment")
		}
	}
}
