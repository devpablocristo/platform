package validate

import (
	"time"
)

// ValidateISODate validates that a string is a valid ISO date (2006-01-02).
func ValidateISODate(field, s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, Err(field, "cannot be empty")
	}

	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return time.Time{}, Err(field, "invalid ISO date format (YYYY-MM-DD)")
	}

	return t, nil
}

// ValidateISOTimestamp validates that a string is a valid ISO timestamp (RFC3339/RFC3339Nano).
func ValidateISOTimestamp(field, s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, Err(field, "cannot be empty")
	}

	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		// Try RFC3339Nano if RFC3339 fails
		t, err = time.Parse(time.RFC3339Nano, s)
		if err != nil {
			return time.Time{}, Err(field, "invalid ISO timestamp format")
		}
	}

	return t, nil
}

// ValidateDateRange validates that start date is before or equal to end date.
// It accepts both ISO dates (YYYY-MM-DD) and ISO timestamps (RFC3339/RFC3339Nano).
func ValidateDateRange(startField, start, endField, end string) error {
	var startTime, endTime time.Time
	var err error

	// Try to parse start as ISO date first, then as timestamp
	startTime, err = ValidateISODate(startField, start)
	if err != nil {
		// If ISO date fails, try as timestamp
		startTime, err = ValidateISOTimestamp(startField, start)
		if err != nil {
			return err
		}
	}

	// Try to parse end as ISO date first, then as timestamp
	endTime, err = ValidateISODate(endField, end)
	if err != nil {
		// If ISO date fails, try as timestamp
		endTime, err = ValidateISOTimestamp(endField, end)
		if err != nil {
			return err
		}
	}

	if startTime.After(endTime) {
		return Err(startField, "must be before or equal to "+endField)
	}

	return nil
}

// ValidateNotFuture validates that an ISO date/time is not in the future (UTC).
func ValidateNotFuture(field, s string) error {
	t, err := ValidateISOTimestamp(field, s)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	if t.After(now) {
		return Err(field, "cannot be in the future")
	}

	return nil
}
