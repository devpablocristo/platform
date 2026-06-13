package paths

import "testing"

func TestSegmentsNonEmpty(t *testing.T) {
	for _, s := range []string{
		SegmentArchived,
		SegmentArchive,
		SegmentUnarchive,
		SegmentTrash,
		SegmentRestore,
		SegmentPurge,
	} {
		if s == "" {
			t.Fatal("empty segment")
		}
	}
}
