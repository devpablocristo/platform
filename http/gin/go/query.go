package ginmw

import (
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/devpablocristo/core/errors/go/domainerr"
)

// ParseOptionalInt64Query parsea un query param opcional a int64.
// Retorna nil si el param no existe, error si es inválido.
func ParseOptionalInt64Query(c *gin.Context, key string) (*int64, error) {
	raw := c.Query(key)
	if raw == "" {
		return nil, nil
	}
	val, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || val <= 0 {
		return nil, domainerr.Validation("invalid " + key)
	}
	return &val, nil
}

// ParseParamID parsea un path param como int64 positivo.
func ParseParamID(c *gin.Context, param string) (int64, error) {
	raw := strings.TrimSpace(c.Param(param))
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, domainerr.Validation("invalid " + param)
	}
	return id, nil
}

// ParseRFC3339 parsea un string RFC3339. Retorna zero time si está vacío.
func ParseRFC3339(raw string) (time.Time, error) {
	if strings.TrimSpace(raw) == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339, strings.TrimSpace(raw))
}

// ParseOptionalRFC3339 parsea un string RFC3339 opcional. Retorna nil si está vacío.
func ParseOptionalRFC3339(raw string) (*time.Time, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(raw))
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

// ParseOptionalRFC3339Ptr parsea un *string RFC3339 opcional.
func ParseOptionalRFC3339Ptr(raw *string) (*time.Time, error) {
	if raw == nil {
		return nil, nil
	}
	return ParseOptionalRFC3339(*raw)
}
