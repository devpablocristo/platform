package validate

import (
	"regexp"
	"strings"
)

var uuid4Regex = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

// ValidateUUID4 validates that a string is a strict UUID v4.
func ValidateUUID4(field, s string) error {
	if s == "" {
		return Err(field, "cannot be empty")
	}

	if !uuid4Regex.MatchString(strings.ToLower(s)) {
		return Err(field, "invalid UUID v4 format")
	}

	// Check variant (bits 6-7 of clock_seq_hi_and_reserved)
	// For UUID v4, variant should be 10xx (binary), which means 8, 9, a, or b
	variant := string(s[19])
	if variant != "8" && variant != "9" && variant != "a" && variant != "b" &&
		variant != "A" && variant != "B" {
		return Err(field, "invalid UUID v4 variant")
	}

	// Check version (bits 4-7 of time_hi_and_version)
	if s[14] != '4' {
		return Err(field, "invalid UUID v4 version")
	}

	return nil
}

// ValidateULID validates that a string is a valid ULID.
func ValidateULID(field, s string) error {
	if s == "" {
		return Err(field, "cannot be empty")
	}

	if len(s) != 26 {
		return Err(field, "must be exactly 26 characters long")
	}

	// Check if all characters are valid Crockford base32
	for _, char := range s {
		if !isValidULIDChar(char) {
			return Err(field, "contains invalid characters")
		}
	}

	return nil
}

// isValidULIDChar checks if a character is valid for ULID (Crockford base32)
func isValidULIDChar(c rune) bool {
	return (c >= '0' && c <= '9') || (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
}
