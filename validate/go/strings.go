package validate

import (
	"fmt"
	"net/mail"
	"net/url"
	"regexp"
	"strings"
	"unicode"
)

// SafeStringRe is the default regex for safe strings
var SafeStringRe = regexp.MustCompile(`^[\p{L}\p{N}\s._-]+$`)

// ValidateRequiredString validates that a string is non-empty after trimming whitespace.
func ValidateRequiredString(field, v string) error {
	if strings.TrimSpace(v) == "" {
		return Err(field, "cannot be empty")
	}
	return nil
}

// ValidateStringLen validates that a string length is within the specified range after trimming.
func ValidateStringLen(field, v string, min, max int) error {
	trimmed := strings.TrimSpace(v)
	length := len(trimmed)

	if length < min {
		return Err(field, fmt.Sprintf("must be at least %d characters long", min))
	}

	if length > max {
		return Err(field, fmt.Sprintf("must be at most %d characters long", max))
	}

	return nil
}

// ValidateSafeString ensures the string matches the provided regex pattern.
func ValidateSafeString(field, v string, re *regexp.Regexp) error {
	if re == nil {
		re = SafeStringRe
	}

	if !re.MatchString(v) {
		return Err(field, "contains invalid characters")
	}

	return nil
}

// ValidateEmail validates that a string is a valid email address.
func ValidateEmail(field, email string) error {
	if email == "" {
		return Err(field, "cannot be empty")
	}

	if strings.Contains(email, " ") {
		return Err(field, "cannot contain spaces")
	}

	if strings.Count(email, "@") != 1 {
		return Err(field, "must contain exactly one @ symbol")
	}

	_, err := mail.ParseAddress(email)
	if err != nil {
		return Err(field, "invalid email format")
	}

	return nil
}

// ValidateURL validates that a string is a valid URL with required scheme and host.
func ValidateURL(field, raw string) error {
	if raw == "" {
		return Err(field, "cannot be empty")
	}

	u, err := url.Parse(raw)
	if err != nil {
		return Err(field, "invalid URL format")
	}

	if u.Scheme == "" {
		return Err(field, "scheme is required")
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return Err(field, "scheme must be http or https")
	}

	if u.Host == "" {
		return Err(field, "host is required")
	}

	return nil
}

// ValidateNumeric validates that a string contains only digits.
func ValidateNumeric(field, s string) error {
	if s == "" {
		return Err(field, "cannot be empty")
	}

	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return Err(field, "cannot be empty after trimming")
	}

	for _, r := range trimmed {
		if !unicode.IsDigit(r) {
			return Err(field, "must contain only digits")
		}
	}

	return nil
}
