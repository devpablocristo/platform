// Package httpjson provides JSON utilities for HTTP services: encoding, decoding, error writing.
package httpjson

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
)

// --- JSON types ---

// APIError es el error HTTP canónico expuesto al cliente.
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type errorEnvelope struct {
	Error APIError `json:"error"`
}

// --- JSON decoding ---

// DecodeJSON decodifica JSON y rechaza campos desconocidos o payload extra.
func DecodeJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	if err := dec.Decode(&struct{}{}); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return err
	}
	return errors.New("unexpected trailing data")
}

// --- JSON response writing ---

// WriteJSON escribe una respuesta JSON.
func WriteJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

// WriteError escribe el envelope HTTP canónico `{error:{code,message}}`.
func WriteError(w http.ResponseWriter, status int, code, message string) {
	WriteJSON(w, status, errorEnvelope{Error: APIError{Code: code, Message: message}})
}

// WriteInternalError loguea el error y escribe un 500 genérico (envelope).
func WriteInternalError(w http.ResponseWriter, err error, ctx string) {
	slog.Error("internal error", "context", ctx, "error", err)
	WriteError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
}

// WriteFlatError escribe un error HTTP con formato plano `{code, message}` (sin envelope).
func WriteFlatError(w http.ResponseWriter, status int, code, message string) {
	WriteJSON(w, status, APIError{Code: code, Message: message})
}

// WriteFlatInternalError loguea el error y escribe un 500 genérico (plano).
func WriteFlatInternalError(w http.ResponseWriter, err error, ctx string) {
	slog.Error("internal error", "context", ctx, "error", err)
	WriteFlatError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
}
