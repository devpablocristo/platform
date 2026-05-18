package validate

import "strings"

// NormalizePagination normalizes page and limit values with defaults.
func NormalizePagination(page, limit int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	return page, limit
}

// ValidatePagination validates that page and limit are within valid ranges.
func ValidatePagination(page, limit int, maxLimit int) error {
	if page < 1 {
		return Err("page", "must be at least 1")
	}

	if limit < 1 {
		return Err("limit", "must be at least 1")
	}

	if limit > maxLimit {
		return Err("limit", "exceeds maximum allowed value")
	}

	return nil
}

// ValidateSort validates that a sort field is in the whitelist.
func ValidateSort(field, sort string, whitelist map[string]struct{}) error {
	if sort == "" {
		return Err(field, "cannot be empty")
	}

	if whitelist != nil {
		if _, exists := whitelist[sort]; !exists {
			return Err(field, "sort field not allowed")
		}
	}

	return nil
}

// ValidateSortDir validates that a sort direction is valid.
func ValidateSortDir(field, dir string) error {
	if dir == "" {
		return Err(field, "cannot be empty")
	}

	dirLower := strings.ToLower(dir)
	if dirLower != "asc" && dirLower != "desc" {
		return Err(field, "must be 'asc' or 'desc'")
	}

	return nil
}
