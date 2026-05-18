package validate

import "strings"

// ValidateCurrencyISO4217 validates that a currency code follows ISO 4217 format.
func ValidateCurrencyISO4217(field, code string, allowed map[string]struct{}) error {
	if code == "" {
		return Err(field, "cannot be empty")
	}

	if len(code) != 3 {
		return Err(field, "must be exactly 3 characters long")
	}

	if code != strings.ToUpper(code) {
		return Err(field, "must be uppercase")
	}

	// Check if code contains only letters
	for _, char := range code {
		if char < 'A' || char > 'Z' {
			return Err(field, "must contain only letters")
		}
	}

	// If allowed currencies are specified, check if code is in the list
	if allowed != nil {
		if _, exists := allowed[code]; !exists {
			return Err(field, "currency code not in allowed list")
		}
	}

	return nil
}

// ValidateMonetaryCents validates that a monetary amount in cents is non-negative.
func ValidateMonetaryCents(field string, cents int64) error {
	if cents < 0 {
		return Err(field, "must be non-negative")
	}
	return nil
}
