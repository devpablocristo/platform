package validate

// ValidateEnumString validates that a string matches one of the allowed values (case-sensitive).
func ValidateEnumString(field, v string, allowed ...string) error {
	if len(allowed) == 0 {
		return Err(field, "no allowed values specified")
	}

	for _, allowedVal := range allowed {
		if v == allowedVal {
			return nil
		}
	}

	return Err(field, "value not in allowed list")
}
