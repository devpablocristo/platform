// Package httperr normaliza errores de dominio a respuestas HTTP.
// Normalize es puro (error → status + APIError).
// Write* son helpers para net/http handlers (delegan en httpjson).
package httperr

import (
	"errors"
	"net/http"

	"github.com/devpablocristo/core/errors/go/domainerr"
	"github.com/devpablocristo/core/http/go/httpjson"
)

// --- Constantes ---

const (
	CodeUnauthorized = "UNAUTHORIZED"
	CodeNotFound     = "NOT_FOUND"
	CodeValidation   = "VALIDATION_ERROR"
	CodeRateLimited  = "RATE_LIMITED"
	CodeInternal     = "INTERNAL"
)

// --- Tipos ---

// APIError reutiliza el tipo canónico de httpjson (un solo shape JSON code+message).
type APIError = httpjson.APIError

// HTTPError es un error que ya sabe su status HTTP.
// Usar solo en la capa de transporte cuando no aplica domainerr.
type HTTPError struct {
	Status  int
	Code    string
	Message string
}

func (e HTTPError) Error() string { return e.Code + ": " + e.Message }

// New crea un HTTPError.
func New(status int, code, message string) HTTPError {
	if status == 0 {
		status = http.StatusInternalServerError
	}
	return HTTPError{Status: status, Code: code, Message: message}
}

// --- Normalización pura (error → status + APIError) ---

var statusByKind = map[domainerr.Kind]int{
	domainerr.KindUnauthorized:  http.StatusUnauthorized,
	domainerr.KindForbidden:     http.StatusForbidden,
	domainerr.KindNotFound:      http.StatusNotFound,
	domainerr.KindValidation:    http.StatusBadRequest,
	domainerr.KindConflict:      http.StatusConflict,
	domainerr.KindBusinessRule:  http.StatusUnprocessableEntity,
	domainerr.KindUnavailable:   http.StatusServiceUnavailable,
	domainerr.KindUpstreamError: http.StatusBadGateway,
	domainerr.KindInternal:      http.StatusInternalServerError,
}

// Normalize mapea un error a (HTTP status, APIError). Función pura, sin I/O.
// Soporta domainerr.Error y HTTPError.
func Normalize(err error) (int, APIError) {
	var de domainerr.Error
	if errors.As(err, &de) {
		status := statusByKind[de.Kind()]
		if status == 0 {
			status = http.StatusInternalServerError
		}
		return status, APIError{Code: string(de.Kind()), Message: de.Message()}
	}

	var he HTTPError
	if errors.As(err, &he) {
		status := he.Status
		if status == 0 {
			status = http.StatusInternalServerError
		}
		return status, APIError{Code: he.Code, Message: he.Message}
	}

	return http.StatusInternalServerError, APIError{Code: CodeInternal, Message: "internal error"}
}

// --- Helpers de escritura HTTP para net/http handlers ---

// WriteJSON escribe JSON al response writer.
func WriteJSON(w http.ResponseWriter, status int, payload any) {
	httpjson.WriteJSON(w, status, payload)
}

// Write escribe un error HTTP con envelope `{error:{code,message}}`.
func Write(w http.ResponseWriter, status int, code, message string) {
	httpjson.WriteError(w, status, code, message)
}

// WriteFrom normaliza un error y lo escribe como respuesta HTTP.
func WriteFrom(w http.ResponseWriter, err error) {
	status, apiErr := Normalize(err)
	Write(w, status, apiErr.Code, apiErr.Message)
}

// BadRequest escribe un error 400.
func BadRequest(w http.ResponseWriter, message string) {
	Write(w, http.StatusBadRequest, CodeValidation, message)
}

// Unauthorized escribe un error 401.
func Unauthorized(w http.ResponseWriter, message string) {
	Write(w, http.StatusUnauthorized, CodeUnauthorized, message)
}

// Forbidden escribe un error 403.
func Forbidden(w http.ResponseWriter, message string) {
	Write(w, http.StatusForbidden, "FORBIDDEN", message)
}

// NotFound escribe un error 404.
func NotFound(w http.ResponseWriter, message string) {
	Write(w, http.StatusNotFound, CodeNotFound, message)
}
