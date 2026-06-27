package httpjson

import (
	"net/http"

	"github.com/devpablocristo/platform/errors/go/domainerr"
)

// StatusForError maps a domainerr-classified error to its canonical HTTP status.
// Unclassified errors map to 500. Agnostic: no app- or business-specific logic.
func StatusForError(err error) int {
	switch {
	case domainerr.IsNotFound(err):
		return http.StatusNotFound
	case domainerr.IsValidation(err):
		return http.StatusBadRequest
	case domainerr.IsForbidden(err):
		return http.StatusForbidden
	case domainerr.IsConflict(err):
		return http.StatusConflict
	case domainerr.IsUnauthorized(err):
		return http.StatusUnauthorized
	default:
		return http.StatusInternalServerError
	}
}

// WriteFlatErrorFrom writes the flat error `{code, message}` derived from a
// domainerr-classified error, so handlers stop hand-rolling the same
// kind→status switch. Client-safe kinds (validation/forbidden/conflict/
// unauthorized) surface err.Error() as the message; NOT_FOUND uses a generic
// "not found" (no leakage); anything unclassified is treated as a 500 — logged
// via WriteFlatInternalError with logCtx, and the client gets a generic message.
//
// Codes are UPPERCASE canonical (NOT_FOUND/VALIDATION/FORBIDDEN/CONFLICT/
// UNAUTHORIZED). Callers needing a different code convention can compose
// StatusForError with WriteFlatError instead.
func WriteFlatErrorFrom(w http.ResponseWriter, err error, logCtx string) {
	switch {
	case domainerr.IsNotFound(err):
		WriteFlatError(w, http.StatusNotFound, "NOT_FOUND", "not found")
	case domainerr.IsValidation(err):
		WriteFlatError(w, http.StatusBadRequest, "VALIDATION", err.Error())
	case domainerr.IsForbidden(err):
		WriteFlatError(w, http.StatusForbidden, "FORBIDDEN", err.Error())
	case domainerr.IsConflict(err):
		WriteFlatError(w, http.StatusConflict, "CONFLICT", err.Error())
	case domainerr.IsUnauthorized(err):
		WriteFlatError(w, http.StatusUnauthorized, "UNAUTHORIZED", err.Error())
	default:
		WriteFlatInternalError(w, err, logCtx)
	}
}
