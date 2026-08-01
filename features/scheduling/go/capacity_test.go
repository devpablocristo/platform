package scheduling

import (
	"testing"
	"time"

	schedulingdomain "github.com/devpablocristo/platform/features/scheduling/go/domain"
	"github.com/google/uuid"
)

func TestNormalizeResourceAllocationsSupportsMultipleResourcesAndUnits(t *testing.T) {
	t.Parallel()

	professionalID := uuid.New()
	roomID := uuid.New()
	got, err := NormalizeResourceAllocations(nil, []schedulingdomain.ResourceAllocation{
		{ResourceID: professionalID, Units: 1},
		{ResourceID: roomID, Units: 2},
		{ResourceID: roomID, Units: 1},
	})
	if err != nil {
		t.Fatalf("normalize allocations: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 allocations, got %d", len(got))
	}
	if got[0].ResourceID != professionalID || got[0].Units != 1 {
		t.Fatalf("unexpected primary allocation: %+v", got[0])
	}
	if got[1].ResourceID != roomID || got[1].Units != 3 {
		t.Fatalf("unexpected room allocation: %+v", got[1])
	}
}

func TestNormalizeResourceAllocationsKeepsLegacyPrimaryAsOneUnit(t *testing.T) {
	t.Parallel()

	resourceID := uuid.New()
	got, err := NormalizeResourceAllocations(&resourceID, nil)
	if err != nil {
		t.Fatalf("normalize primary: %v", err)
	}
	if len(got) != 1 || got[0].ResourceID != resourceID || got[0].Units != 1 {
		t.Fatalf("unexpected allocation: %+v", got)
	}
}

func TestNormalizeResourceAllocationsDefaultsCapacityUnitsFromParticipants(t *testing.T) {
	t.Parallel()

	resourceID := uuid.New()
	got, err := NormalizeResourceAllocationsForParticipants(nil, []schedulingdomain.ResourceAllocation{{
		ResourceID: resourceID,
	}}, 4)
	if err != nil {
		t.Fatalf("normalize participant allocation: %v", err)
	}
	if len(got) != 1 || got[0].Mode != schedulingdomain.ResourceAllocationModeCapacity || got[0].Units != 4 {
		t.Fatalf("unexpected participant allocation: %+v", got)
	}
}

func TestResolveAllocationUnitsExpandsExclusiveAllocation(t *testing.T) {
	t.Parallel()

	resourceID := uuid.New()
	got, err := ResolveAllocationUnits(schedulingdomain.ResourceAllocation{
		ResourceID: resourceID,
		Mode:       schedulingdomain.ResourceAllocationModeExclusive,
		Units:      1,
	}, 12)
	if err != nil {
		t.Fatalf("resolve exclusive allocation: %v", err)
	}
	if got.Mode != schedulingdomain.ResourceAllocationModeExclusive || got.Units != 12 {
		t.Fatalf("exclusive allocation = %+v, want all 12 units", got)
	}
}

func TestNormalizeResourceAllocationsRejectsMixedDuplicateModes(t *testing.T) {
	t.Parallel()

	resourceID := uuid.New()
	_, err := NormalizeResourceAllocations(nil, []schedulingdomain.ResourceAllocation{
		{ResourceID: resourceID, Mode: schedulingdomain.ResourceAllocationModeCapacity, Units: 1},
		{ResourceID: resourceID, Mode: schedulingdomain.ResourceAllocationModeExclusive},
	})
	if err == nil {
		t.Fatal("expected duplicate allocation mode conflict")
	}
}

func TestRemainingReservationsAccountsForRequestedUnits(t *testing.T) {
	t.Parallel()

	if got := RemainingReservations(8, 2, 2); got != 3 {
		t.Fatalf("remaining reservations = %d, want 3", got)
	}
	if got := RemainingReservations(2, 2, 1); got != 0 {
		t.Fatalf("full resource remaining = %d, want 0", got)
	}
}

func TestIntersectResourceSlotsRequiresEveryResourceAtSameRange(t *testing.T) {
	t.Parallel()

	professionalID := uuid.New()
	roomID := uuid.New()
	start := time.Date(2026, 8, 3, 13, 0, 0, 0, time.UTC)
	end := start.Add(45 * time.Minute)
	occupiesFrom := start.Add(-10 * time.Minute)
	occupiesUntil := end.Add(5 * time.Minute)

	request := []schedulingdomain.ResourceAllocation{
		{ResourceID: professionalID, Mode: schedulingdomain.ResourceAllocationModeCapacity, Units: 1},
		{ResourceID: roomID, Mode: schedulingdomain.ResourceAllocationModeCapacity, Units: 2},
	}
	slotSets := map[uuid.UUID][]schedulingdomain.TimeSlot{
		professionalID: {
			{
				ResourceID:       professionalID,
				ResourceName:     "Ana",
				StartAt:          start,
				EndAt:            end,
				OccupiesFrom:     occupiesFrom,
				OccupiesUntil:    occupiesUntil,
				Remaining:        3,
				ServiceRemaining: 4,
				AllocatedUnits:   1,
			},
		},
		roomID: {
			{
				ResourceID:       roomID,
				ResourceName:     "Sala 1",
				StartAt:          start,
				EndAt:            end,
				OccupiesFrom:     occupiesFrom,
				OccupiesUntil:    occupiesUntil,
				Remaining:        4,
				ServiceRemaining: 4,
				AllocatedUnits:   2,
			},
		},
	}

	got := IntersectResourceSlots(request, slotSets)
	if len(got) != 1 {
		t.Fatalf("expected one composite slot, got %d", len(got))
	}
	if len(got[0].Allocations) != 2 {
		t.Fatalf("expected two allocations, got %+v", got[0].Allocations)
	}
	if got[0].Remaining != 2 {
		t.Fatalf("remaining composite reservations = %d, want 2", got[0].Remaining)
	}
	if got[0].AllocatedUnits != 3 {
		t.Fatalf("allocated units = %d, want 3", got[0].AllocatedUnits)
	}
}

func TestIntersectResourceSlotsRejectsCapacityOrRangeMismatch(t *testing.T) {
	t.Parallel()

	leftID := uuid.New()
	rightID := uuid.New()
	start := time.Date(2026, 8, 3, 13, 0, 0, 0, time.UTC)
	slot := schedulingdomain.TimeSlot{
		StartAt:          start,
		EndAt:            start.Add(time.Hour),
		OccupiesFrom:     start,
		OccupiesUntil:    start.Add(time.Hour),
		Remaining:        1,
		ServiceRemaining: 1,
	}

	request := []schedulingdomain.ResourceAllocation{
		{ResourceID: leftID, Mode: schedulingdomain.ResourceAllocationModeCapacity, Units: 1},
		{ResourceID: rightID, Mode: schedulingdomain.ResourceAllocationModeCapacity, Units: 2},
	}
	slotSets := map[uuid.UUID][]schedulingdomain.TimeSlot{
		leftID:  {slot},
		rightID: {slot},
	}
	if got := IntersectResourceSlots(request, slotSets); len(got) != 0 {
		t.Fatalf("expected capacity mismatch to reject slot, got %+v", got)
	}

	right := slot
	right.Remaining = 2
	right.StartAt = right.StartAt.Add(time.Minute)
	slotSets[rightID] = []schedulingdomain.TimeSlot{right}
	if got := IntersectResourceSlots(request, slotSets); len(got) != 0 {
		t.Fatalf("expected range mismatch to reject slot, got %+v", got)
	}
}

func TestIntersectResourceSlotsRequiresEmptyResourceForExclusiveAllocation(t *testing.T) {
	t.Parallel()

	resourceID := uuid.New()
	start := time.Date(2026, 8, 3, 13, 0, 0, 0, time.UTC)
	slot := schedulingdomain.TimeSlot{
		ResourceID:       resourceID,
		StartAt:          start,
		EndAt:            start.Add(time.Hour),
		OccupiesFrom:     start,
		OccupiesUntil:    start.Add(time.Hour),
		Remaining:        7,
		ServiceRemaining: 2,
		AllocatedUnits:   1,
	}
	request := []schedulingdomain.ResourceAllocation{{
		ResourceID: resourceID,
		Mode:       schedulingdomain.ResourceAllocationModeExclusive,
		Units:      8,
	}}
	if got := IntersectResourceSlots(request, map[uuid.UUID][]schedulingdomain.TimeSlot{resourceID: {slot}}); len(got) != 0 {
		t.Fatalf("exclusive allocation accepted an occupied resource: %+v", got)
	}

	slot.AllocatedUnits = 0
	slot.Remaining = 8
	got := IntersectResourceSlots(request, map[uuid.UUID][]schedulingdomain.TimeSlot{resourceID: {slot}})
	if len(got) != 1 || got[0].Remaining != 1 {
		t.Fatalf("exclusive allocation = %+v, want one available reservation", got)
	}
}
