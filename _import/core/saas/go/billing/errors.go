package billing

import (
	"net/http"

	"github.com/devpablocristo/core/http/go/httperr"
)

const (
	ErrCodeUnauthorized = httperr.CodeUnauthorized
	ErrCodeNotFound     = httperr.CodeNotFound
	ErrCodeValidation   = httperr.CodeValidation
	ErrCodeInternal     = httperr.CodeInternal
	ErrCodeRateLimit    = httperr.CodeRateLimited
)

type HTTPError = httperr.HTTPError

func NewHTTPError(status int, code, message string) HTTPError {
	return httperr.New(status, code, message)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	httperr.WriteJSON(w, status, payload)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	httperr.Write(w, status, code, message)
}

func writeErrorFrom(w http.ResponseWriter, err error) {
	httperr.WriteFrom(w, err)
}

func writeBadRequest(w http.ResponseWriter, message string) {
	httperr.BadRequest(w, message)
}
