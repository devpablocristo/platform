package validate

import (
	"fmt"
)

// ValidateIntRange validates that an integer is within the specified range.
func ValidateIntRange(field string, n, min, max int64) error {
	if n < min || n > max {
		return Err(field, fmt.Sprintf("must be between %d and %d", min, max))
	}
	return nil
}

// ValidateFloatRange validates that a float is within the specified range.
func ValidateFloatRange(field string, n, min, max float64) error {
	if n < min || n > max {
		return Err(field, fmt.Sprintf("must be between %f and %f", min, max))
	}
	return nil
}

// ValidateNonNegative validates that an integer is non-negative.
func ValidateNonNegative(field string, n int64) error {
	if n < 0 {
		return Err(field, "must be non-negative")
	}
	return nil
}
