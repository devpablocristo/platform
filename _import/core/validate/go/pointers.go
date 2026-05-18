package validate

// RequireNotNil validates that a pointer is not nil.
func RequireNotNil[T any](field string, ptr *T) error {
	if ptr == nil {
		return Err(field, "cannot be nil")
	}
	return nil
}

// ValidateOptional validates a pointer only if it's not nil.
func ValidateOptional[T any](ptr *T, f func(v T) error) error {
	if ptr == nil {
		return nil
	}
	return f(*ptr)
}
