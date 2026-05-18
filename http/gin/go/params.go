package ginmw

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ParseUUIDParam parsea un UUID de un parámetro de URL de Gin.
// Escribe error 400 y retorna false si es inválido.
func ParseUUIDParam(c *gin.Context, param string) (uuid.UUID, bool) {
	value := strings.TrimSpace(c.Param(param))
	id, err := uuid.Parse(value)
	if err != nil {
		c.JSON(http.StatusBadRequest, SimpleErrorResponse{Error: "invalid " + param})
		return uuid.Nil, false
	}
	return id, true
}

// ParseUUIDQuery parsea un UUID de un query parameter.
// Retorna uuid.Nil, true si está vacío (optional).
func ParseUUIDQuery(c *gin.Context, param string) (uuid.UUID, bool) {
	raw := strings.TrimSpace(c.Query(param))
	if raw == "" {
		return uuid.Nil, true
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		c.JSON(http.StatusBadRequest, SimpleErrorResponse{Error: "invalid " + param})
		return uuid.Nil, false
	}
	return id, true
}

// ParseLimitQuery parsea un query param de paginación "limit".
// Retorna defaultLimit si vacío, error 400 si inválido.
func ParseLimitQuery(c *gin.Context, defaultLimit, maxLimit int) (int, bool) {
	raw := strings.TrimSpace(c.Query("limit"))
	if raw == "" {
		return defaultLimit, true
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		c.JSON(http.StatusBadRequest, SimpleErrorResponse{Error: "invalid limit"})
		return 0, false
	}
	if maxLimit > 0 && n > maxLimit {
		n = maxLimit
	}
	return n, true
}

// ParsePageQuery parsea query params de paginación offset-based (page, per_page).
func ParsePageQuery(c *gin.Context, defaultPage, defaultPerPage int) (page, perPage int) {
	page = defaultPage
	perPage = defaultPerPage

	if raw := c.Query("page"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n >= 1 {
			page = n
		}
	}
	if raw := c.Query("per_page"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n >= 1 {
			perPage = n
		}
	} else if raw := c.Query("page_size"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n >= 1 {
			perPage = n
		}
	}
	return page, perPage
}
