package scheduling

import (
	"fmt"
	"sort"
	"strings"
	"time"

	schedulingdomain "github.com/devpablocristo/platform/features/scheduling/go/domain"
	"github.com/google/uuid"
)

const maxAllocationUnits = 100_000

// NormalizeResourceAllocations validates, deduplicates, and stabilizes a
// resource request. The legacy primary resource is interpreted as one unit
// only when no explicit allocations were supplied.
func NormalizeResourceAllocations(
	primary *uuid.UUID,
	allocations []schedulingdomain.ResourceAllocation,
) ([]schedulingdomain.ResourceAllocation, error) {
	return NormalizeResourceAllocationsForParticipants(primary, allocations, 1)
}

// NormalizeResourceAllocationsForParticipants applies participant-aware
// defaults. Capacity allocations with zero Units consume one unit per
// participant; exclusive allocations are later expanded to the resource's
// complete capacity by ResolveAllocationUnits.
func NormalizeResourceAllocationsForParticipants(
	primary *uuid.UUID,
	allocations []schedulingdomain.ResourceAllocation,
	participants int,
) ([]schedulingdomain.ResourceAllocation, error) {
	if participants <= 0 {
		return nil, fmt.Errorf("participants must be positive")
	}
	if participants > maxAllocationUnits {
		return nil, fmt.Errorf("participants must be <= %d", maxAllocationUnits)
	}
	if len(allocations) == 0 && primary != nil && *primary != uuid.Nil {
		allocations = []schedulingdomain.ResourceAllocation{{
			ResourceID: *primary,
			Mode:       schedulingdomain.ResourceAllocationModeCapacity,
			Units:      participants,
		}}
	}
	if len(allocations) == 0 {
		return nil, nil
	}

	order := make([]uuid.UUID, 0, len(allocations))
	normalized := make(map[uuid.UUID]schedulingdomain.ResourceAllocation, len(allocations))
	for _, allocation := range allocations {
		if allocation.ResourceID == uuid.Nil {
			return nil, fmt.Errorf("resource_id is required")
		}
		mode, err := normalizeResourceAllocationMode(allocation.Mode)
		if err != nil {
			return nil, err
		}
		allocation.Mode = mode
		if allocation.Units < 0 {
			return nil, fmt.Errorf("resource allocation units must not be negative")
		}
		if allocation.Mode == schedulingdomain.ResourceAllocationModeCapacity && allocation.Units == 0 {
			allocation.Units = participants
		}
		if allocation.Mode == schedulingdomain.ResourceAllocationModeExclusive {
			allocation.Units = 1
		}
		current, exists := normalized[allocation.ResourceID]
		if !exists {
			order = append(order, allocation.ResourceID)
			current = schedulingdomain.ResourceAllocation{
				ResourceID:   allocation.ResourceID,
				ResourceName: strings.TrimSpace(allocation.ResourceName),
				Mode:         allocation.Mode,
			}
		} else if current.Mode != allocation.Mode {
			return nil, fmt.Errorf("resource allocation mode must be consistent for duplicate resources")
		}
		if allocation.Mode == schedulingdomain.ResourceAllocationModeExclusive {
			current.Units = 1
		} else {
			current.Units += allocation.Units
		}
		if current.Units > maxAllocationUnits {
			return nil, fmt.Errorf("resource allocation units must be <= %d", maxAllocationUnits)
		}
		if current.ResourceName == "" {
			current.ResourceName = strings.TrimSpace(allocation.ResourceName)
		}
		normalized[allocation.ResourceID] = current
	}

	out := make([]schedulingdomain.ResourceAllocation, 0, len(order))
	for _, resourceID := range order {
		out = append(out, normalized[resourceID])
	}
	return out, nil
}

// ResolveAllocationUnits expands an allocation into the units the storage
// adapter must reserve atomically.
func ResolveAllocationUnits(
	allocation schedulingdomain.ResourceAllocation,
	resourceCapacity int,
) (schedulingdomain.ResourceAllocation, error) {
	if resourceCapacity <= 0 {
		return schedulingdomain.ResourceAllocation{}, fmt.Errorf("resource capacity must be positive")
	}
	mode, err := normalizeResourceAllocationMode(allocation.Mode)
	if err != nil {
		return schedulingdomain.ResourceAllocation{}, err
	}
	allocation.Mode = mode
	if mode == schedulingdomain.ResourceAllocationModeExclusive {
		allocation.Units = resourceCapacity
	}
	if allocation.Units <= 0 {
		return schedulingdomain.ResourceAllocation{}, fmt.Errorf("resource allocation units must be positive")
	}
	if allocation.Units > resourceCapacity {
		return schedulingdomain.ResourceAllocation{}, fmt.Errorf("requested units exceed resource capacity")
	}
	return allocation, nil
}

func normalizeResourceAllocationMode(
	mode schedulingdomain.ResourceAllocationMode,
) (schedulingdomain.ResourceAllocationMode, error) {
	switch schedulingdomain.ResourceAllocationMode(strings.ToLower(strings.TrimSpace(string(mode)))) {
	case "", schedulingdomain.ResourceAllocationModeCapacity:
		return schedulingdomain.ResourceAllocationModeCapacity, nil
	case schedulingdomain.ResourceAllocationModeExclusive:
		return schedulingdomain.ResourceAllocationModeExclusive, nil
	default:
		return "", fmt.Errorf("invalid resource allocation mode")
	}
}

// RemainingReservations reports how many additional reservations requesting
// units can fit in a resource snapshot.
func RemainingReservations(capacity, allocated, units int) int {
	if capacity <= 0 || units <= 0 || allocated >= capacity {
		return 0
	}
	remainingUnits := capacity - max(allocated, 0)
	return remainingUnits / units
}

// IntersectResourceSlots produces only the ranges in which every requested
// resource has enough capacity. Slot sets must already include availability,
// blocks, external busy periods, and current allocations for each resource.
func IntersectResourceSlots(
	request []schedulingdomain.ResourceAllocation,
	slotSets map[uuid.UUID][]schedulingdomain.TimeSlot,
) []schedulingdomain.TimeSlot {
	if len(request) == 0 {
		return nil
	}
	for index := range request {
		mode, err := normalizeResourceAllocationMode(request[index].Mode)
		if err != nil {
			return nil
		}
		request[index].Mode = mode
	}

	primarySlots := slotSets[request[0].ResourceID]
	out := make([]schedulingdomain.TimeSlot, 0, len(primarySlots))
	for _, primary := range primarySlots {
		allocations := make([]schedulingdomain.ResourceAllocation, 0, len(request))
		remaining := 0
		serviceRemaining := 0
		allocatedUnits := 0
		matched := true

		for index, requested := range request {
			slot, ok := findMatchingSlot(slotSets[requested.ResourceID], primary)
			if !ok {
				matched = false
				break
			}
			resourceName := strings.TrimSpace(requested.ResourceName)
			if resourceName == "" {
				resourceName = slot.ResourceName
			}
			allocations = append(allocations, schedulingdomain.ResourceAllocation{
				ResourceID:   requested.ResourceID,
				ResourceName: resourceName,
				Mode:         requested.Mode,
				Units:        requested.Units,
			})
			resourceRemaining := 0
			if requested.Mode == schedulingdomain.ResourceAllocationModeExclusive {
				if slot.AllocatedUnits > 0 {
					matched = false
					break
				}
				resourceRemaining = 1
			} else {
				if requested.Units <= 0 || slot.Remaining < requested.Units {
					matched = false
					break
				}
				resourceRemaining = slot.Remaining / requested.Units
			}
			if slot.ServiceRemaining > 0 && slot.ServiceRemaining < resourceRemaining {
				resourceRemaining = slot.ServiceRemaining
			}
			if index == 0 || resourceRemaining < remaining {
				remaining = resourceRemaining
			}
			if index == 0 || (slot.ServiceRemaining > 0 && slot.ServiceRemaining < serviceRemaining) {
				serviceRemaining = slot.ServiceRemaining
			}
			allocatedUnits += slot.AllocatedUnits
		}
		if !matched || remaining <= 0 {
			continue
		}

		composite := primary
		composite.Allocations = allocations
		composite.ResourceID = allocations[0].ResourceID
		composite.ResourceName = allocations[0].ResourceName
		composite.Remaining = remaining
		composite.ServiceRemaining = serviceRemaining
		composite.AllocatedUnits = allocatedUnits
		out = append(out, composite)
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].StartAt.Equal(out[j].StartAt) {
			return out[i].ResourceName < out[j].ResourceName
		}
		return out[i].StartAt.Before(out[j].StartAt)
	})
	return out
}

func findMatchingSlot(
	slots []schedulingdomain.TimeSlot,
	target schedulingdomain.TimeSlot,
) (schedulingdomain.TimeSlot, bool) {
	for _, slot := range slots {
		if sameInstant(slot.StartAt, target.StartAt) &&
			sameInstant(slot.EndAt, target.EndAt) &&
			sameInstant(slot.OccupiesFrom, target.OccupiesFrom) &&
			sameInstant(slot.OccupiesUntil, target.OccupiesUntil) {
			return slot, true
		}
	}
	return schedulingdomain.TimeSlot{}, false
}

func sameInstant(left, right time.Time) bool {
	return left.UTC().Equal(right.UTC())
}
