package ginmw

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/devpablocristo/core/errors/go/domainerr"
	"github.com/devpablocristo/core/http/go/httperr"
)

// Sentinel errors reutilizables para handlers Gin de producto.
var (
	ErrNotFound  = domainerr.NotFound("not found")
	ErrConflict  = domainerr.Conflict("conflict")
	ErrForbidden = domainerr.Forbidden("forbidden")
	ErrBadInput  = domainerr.Validation("bad input")
)

// --- Tipos de respuesta HTTP (inline, no sub-paquete dto/) ---

// ErrorResponse es el envelope de error HTTP estándar para domain/app errors.
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Meta    any    `json:"meta,omitempty"`
}

// SimpleErrorResponse respuesta de error con un solo campo message.
// Usado por middleware (rate limit, auth, body limit, params).
type SimpleErrorResponse struct {
	Error string `json:"error"`
}

// HealthResponse respuesta de health/readiness check.
type HealthResponse struct {
	Status string `json:"status"`
}

// --- Response writers ---

// Respond mapea un error de dominio/transporte a respuesta JSON en Gin.
func Respond(c *gin.Context, err error) {
	var de domainerr.Error
	if errors.As(err, &de) {
		status, apiErr := httperr.Normalize(err)
		c.JSON(status, ErrorResponse{Code: apiErr.Code, Message: apiErr.Message})
		return
	}

	var he httperr.HTTPError
	if errors.As(err, &he) {
		st := he.Status
		if st == 0 {
			st = http.StatusInternalServerError
		}
		c.JSON(st, ErrorResponse{Code: he.Code, Message: he.Message})
		return
	}

	// Fallback — NUNCA exponer err.Error() al cliente
	slog.Error("unhandled error in http response", "error", err)
	c.JSON(http.StatusInternalServerError, ErrorResponse{Code: "internal_error", Message: "unexpected error"})
}

// WriteError escribe un error HTTP genérico a Gin.
func WriteError(c *gin.Context, status int, code, message string) {
	c.JSON(status, ErrorResponse{Code: code, Message: message})
}

// WriteJSON escribe una respuesta JSON exitosa.
func WriteJSON(c *gin.Context, status int, data any) {
	c.JSON(status, data)
}

// WriteCreated escribe un 201 con el payload.
func WriteCreated(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, data)
}

// WriteNoContent escribe un 204.
func WriteNoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}
