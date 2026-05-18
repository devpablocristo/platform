// Package scheduling exposes reusable scheduling primitives for slot generation.
package scheduling

import (
	"strings"
	"time"
)

// Window defines a local-time availability interval.
type Window struct {
	Start              time.Time
	End                time.Time
	GranularityMinutes int
}

// BlockedRange defines a time interval that cannot be occupied.
type BlockedRange struct {
	StartAt time.Time
	EndAt   time.Time
}

// SlotSpec controls duration, buffers and default slot granularity.
type SlotSpec struct {
	DurationMinutes           int
	BufferBeforeMinutes       int
	BufferAfterMinutes        int
	DefaultGranularityMinutes int
}

// Slot is a generated candidate slot.
type Slot struct {
	StartAt            time.Time
	EndAt              time.Time
	OccupiesFrom       time.Time
	OccupiesUntil      time.Time
	GranularityMinutes int
}

// ParseClock parses a clock string in HH:MM or HH:MM:SS form.
// Both shapes appear in the wild: domain code stores "HH:MM" while Postgres
// `time` columns are read by GORM as "HH:MM:SS".
func ParseClock(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if t, err := time.Parse("15:04:05", raw); err == nil {
		return t, nil
	}
	return time.Parse("15:04", raw)
}

// IntersectWindows intersects base windows with overlay windows.
// If one side is empty, the other side is considered authoritative.
func IntersectWindows(base, overlay []Window) []Window {
	if len(base) == 0 && len(overlay) == 0 {
		return nil
	}
	if len(base) == 0 {
		return cloneWindows(overlay)
	}
	if len(overlay) == 0 {
		return cloneWindows(base)
	}
	out := make([]Window, 0, len(base)*len(overlay))
	for _, bw := range base {
		for _, ow := range overlay {
			start := maxTime(bw.Start, ow.Start)
			end := minTime(bw.End, ow.End)
			if !end.After(start) {
				continue
			}
			granularity := ow.GranularityMinutes
			if granularity <= 0 {
				granularity = bw.GranularityMinutes
			}
			out = append(out, Window{
				Start:              start,
				End:                end,
				GranularityMinutes: granularity,
			})
		}
	}
	return out
}

// OverlapsBlocked reports whether a candidate occupancy overlaps blocked ranges.
func OverlapsBlocked(startAt, endAt time.Time, blocked []BlockedRange) bool {
	for _, block := range blocked {
		if startAt.Before(block.EndAt.UTC()) && endAt.After(block.StartAt.UTC()) {
			return true
		}
	}
	return false
}

// GenerateSlots expands windows into candidate slots using the supplied spec.
func GenerateSlots(windows []Window, blocked []BlockedRange, spec SlotSpec) []Slot {
	duration := time.Duration(spec.DurationMinutes) * time.Minute
	if duration <= 0 {
		return nil
	}
	bufferBefore := time.Duration(spec.BufferBeforeMinutes) * time.Minute
	bufferAfter := time.Duration(spec.BufferAfterMinutes) * time.Minute

	slots := make([]Slot, 0)
	for _, window := range windows {
		step := window.GranularityMinutes
		if step <= 0 {
			step = spec.DefaultGranularityMinutes
		}
		if step <= 0 {
			step = 15
		}
		for cursor := window.Start; !cursor.Add(duration).After(window.End); cursor = cursor.Add(time.Duration(step) * time.Minute) {
			endAt := cursor.Add(duration)
			occupiesFrom := cursor.Add(-bufferBefore)
			occupiesUntil := endAt.Add(bufferAfter)
			if OverlapsBlocked(occupiesFrom.UTC(), occupiesUntil.UTC(), blocked) {
				continue
			}
			slots = append(slots, Slot{
				StartAt:            cursor.UTC(),
				EndAt:              endAt.UTC(),
				OccupiesFrom:       occupiesFrom.UTC(),
				OccupiesUntil:      occupiesUntil.UTC(),
				GranularityMinutes: step,
			})
		}
	}
	return slots
}

func cloneWindows(in []Window) []Window {
	out := make([]Window, len(in))
	copy(out, in)
	return out
}

func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}
