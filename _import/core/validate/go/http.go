package validate

import (
	"regexp"
	"strings"
)

var idempotencyKeyRegex = regexp.MustCompile(`^[^\s]{8,128}$`)

// ValidateContentType valida que el content type esté en la lista permitida (valor de cabecera ya extraído).
func ValidateContentType(field, ct string, allowed ...string) error {
	if ct == "" {
		return Err(field, "cannot be empty")
	}

	// Strip parameters (everything after ;)
	baseType := strings.Split(ct, ";")[0]
	baseType = strings.ToLower(strings.TrimSpace(baseType))

	if len(allowed) == 0 {
		return nil // No restrictions
	}

	for _, allowedType := range allowed {
		if strings.ToLower(strings.TrimSpace(allowedType)) == baseType {
			return nil
		}
	}

	return Err(field, "content type not allowed")
}

// ValidateBearerFormat validates that an authorization header follows Bearer format.
func ValidateBearerFormat(field, auth string) (string, error) {
	if auth == "" {
		return "", Err(field, "cannot be empty")
	}

	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return "", Err(field, "must be in format 'Bearer <token>'")
	}

	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", Err(field, "token cannot be empty")
	}

	return token, nil
}

// ValidateIdempotencyKey validates that an idempotency key follows the required format.
func ValidateIdempotencyKey(field, k string) error {
	if k == "" {
		return Err(field, "cannot be empty")
	}

	if !idempotencyKeyRegex.MatchString(k) {
		return Err(field, "must be 8-128 characters with no whitespace")
	}

	return nil
}
