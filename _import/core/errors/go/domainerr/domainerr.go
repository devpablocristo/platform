package domainerr

import (
	"errors"
	"fmt"
)

// Kind categoriza errores de dominio sin acoplarlos a transporte.
type Kind string

const (
	KindUnauthorized  Kind = "UNAUTHORIZED"
	KindForbidden     Kind = "FORBIDDEN"
	KindNotFound      Kind = "NOT_FOUND"
	KindValidation    Kind = "VALIDATION_ERROR"
	KindConflict      Kind = "CONFLICT"
	KindBusinessRule  Kind = "BUSINESS_RULE"
	KindUnavailable   Kind = "UNAVAILABLE"
	KindUpstreamError Kind = "UPSTREAM_ERROR"
	KindInternal      Kind = "INTERNAL"
)

// Error representa un error de dominio categorizado.
type Error struct {
	kind    Kind
	message string
}

func (e Error) Error() string { return string(e.kind) + ": " + e.message }

func (e Error) Kind() Kind { return e.kind }

func (e Error) Message() string { return e.message }

func New(kind Kind, message string) Error {
	return Error{kind: kind, message: message}
}

func Newf(kind Kind, format string, args ...any) Error {
	return Error{kind: kind, message: fmt.Sprintf(format, args...)}
}

func Unauthorized(message string) Error  { return New(KindUnauthorized, message) }
func Forbidden(message string) Error     { return New(KindForbidden, message) }
func NotFound(message string) Error      { return New(KindNotFound, message) }
func Validation(message string) Error    { return New(KindValidation, message) }
func Conflict(message string) Error      { return New(KindConflict, message) }
func BusinessRule(message string) Error  { return New(KindBusinessRule, message) }
func Unavailable(message string) Error   { return New(KindUnavailable, message) }
func UpstreamError(message string) Error { return New(KindUpstreamError, message) }
func Internal(message string) Error      { return New(KindInternal, message) }

// NotFoundf construye un NOT_FOUND con formato (reemplaza apperror.NewNotFound(resource, id)).
func NotFoundf(resource, id string) Error {
	if id == "" {
		return NotFound(resource + " not found")
	}
	return Newf(KindNotFound, "%s '%s' not found", resource, id)
}

// Is compara por Kind (no por mensaje). Permite errors.Is(err, domainerr.NotFound("")).
func (e Error) Is(target error) bool {
	typed, ok := target.(Error)
	if !ok {
		return false
	}
	return e.kind == typed.kind
}

// --- Helpers para chequear kind sin saber el tipo ---

// IsNotFound verifica si un error es de tipo NOT_FOUND.
func IsNotFound(err error) bool { return IsKind(err, KindNotFound) }

// IsConflict verifica si un error es de tipo CONFLICT.
func IsConflict(err error) bool { return IsKind(err, KindConflict) }

// IsValidation verifica si un error es de tipo VALIDATION_ERROR.
func IsValidation(err error) bool { return IsKind(err, KindValidation) }

// IsForbidden verifica si un error es de tipo FORBIDDEN.
func IsForbidden(err error) bool { return IsKind(err, KindForbidden) }

// IsUnauthorized verifica si un error es de tipo UNAUTHORIZED.
func IsUnauthorized(err error) bool { return IsKind(err, KindUnauthorized) }

// IsKind verifica si un error (o su cadena de wrapping) es un domainerr con el Kind dado.
func IsKind(err error, kind Kind) bool {
	var de Error
	if errors.As(err, &de) {
		return de.kind == kind
	}
	return false
}
