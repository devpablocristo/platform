package scheduling

import (
	"testing"
	"time"
)

func TestGenerateSlotsAppliesIntersectionAndBlockedRanges(t *testing.T) {
	t.Parallel()

	loc, err := time.LoadLocation("America/Argentina/Tucuman")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}
	day := time.Date(2026, 4, 6, 0, 0, 0, 0, loc)
	base := []Window{{
		Start:              time.Date(day.Year(), day.Month(), day.Day(), 9, 0, 0, 0, loc),
		End:                time.Date(day.Year(), day.Month(), day.Day(), 12, 0, 0, 0, loc),
		GranularityMinutes: 30,
	}}
	overlay := []Window{{
		Start:              time.Date(day.Year(), day.Month(), day.Day(), 10, 0, 0, 0, loc),
		End:                time.Date(day.Year(), day.Month(), day.Day(), 12, 0, 0, 0, loc),
		GranularityMinutes: 30,
	}}
	blocked := []BlockedRange{{
		StartAt: time.Date(day.Year(), day.Month(), day.Day(), 10, 30, 0, 0, loc).UTC(),
		EndAt:   time.Date(day.Year(), day.Month(), day.Day(), 11, 0, 0, 0, loc).UTC(),
	}}

	slots := GenerateSlots(IntersectWindows(base, overlay), blocked, SlotSpec{
		DurationMinutes:           30,
		DefaultGranularityMinutes: 30,
	})

	if len(slots) != 3 {
		t.Fatalf("expected 3 slots, got %d", len(slots))
	}
	if got := slots[0].StartAt.In(loc).Format("15:04"); got != "10:00" {
		t.Fatalf("first slot = %s, want 10:00", got)
	}
	if got := slots[1].StartAt.In(loc).Format("15:04"); got != "11:00" {
		t.Fatalf("second slot = %s, want 11:00", got)
	}
	if got := slots[2].StartAt.In(loc).Format("15:04"); got != "11:30" {
		t.Fatalf("third slot = %s, want 11:30", got)
	}
}
