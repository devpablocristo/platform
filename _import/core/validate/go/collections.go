package validate

import (
	"fmt"
	"strings"
)

// ValidateStringSliceNotEmpty validates that a string slice is not empty.
func ValidateStringSliceNotEmpty(field string, xs []string) error {
	if len(xs) == 0 {
		return Err(field, "cannot be empty")
	}
	return nil
}

// ValidateUniqueStrings validates that all strings in a slice are unique.
func ValidateUniqueStrings(field string, xs []string, caseInsensitive bool) error {
	if len(xs) <= 1 {
		return nil
	}

	seen := make(map[string]bool)
	for _, s := range xs {
		key := s
		if caseInsensitive {
			key = strings.ToLower(s)
		}

		if seen[key] {
			return Err(field, "contains duplicate values")
		}
		seen[key] = true
	}

	return nil
}

// ValidateSliceLen validates that a slice length is within the specified range.
func ValidateSliceLen(field string, n, min, max int) error {
	if n < min {
		return Err(field, fmt.Sprintf("must have at least %d items", min))
	}

	if n > max {
		return Err(field, fmt.Sprintf("must have at most %d items", max))
	}

	return nil
}
