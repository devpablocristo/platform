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
//
// Serializa ANTES de escribir la cabecera. Al revés —WriteHeader y después Encode— un
// payload que no serializa deja el status ya emitido y el cuerpo truncado: el cliente
// recibe un 200 con JSON inválido y no hay forma de corregirlo, porque la cabecera no se
// puede reescribir. Serializando primero, ese caso se convierte en el 500 que realmente es.
func WriteJSON(w http.ResponseWriter, status int, payload any) {
	body, err := json.Marshal(payload)
	if err != nil {
		slog.Error("response payload could not be encoded", "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		// Literal y no otro Marshal: este cuerpo no puede fallar, y si el fallback
		// dependiera de serializar algo volveríamos al mismo problema.
		_, _ = w.Write([]byte("{\"error\":{\"code\":\"INTERNAL\",\"message\":\"internal error\"}}\n"))
		return
	}
	// El salto final se conserva: es lo que agregaba `Encode`, y todo consumidor de
	// platform ya recibe ese byte. Cambiarlo de paso convertiría un arreglo de status en
	// una modificación del cuerpo de cada respuesta del ecosistema.
	body = append(body, '\n')

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err := w.Write(body); err != nil {
		// La cabecera ya salió, así que el status no se puede cambiar. Se loguea porque
		// una respuesta truncada es información real y descartarla la haría invisible.
		slog.Warn("response body could not be written", "error", err)
	}
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
