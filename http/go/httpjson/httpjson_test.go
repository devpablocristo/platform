package httpjson

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecodeJSONRejectsTrailingData(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodPost, "/v1/resources", strings.NewReader(`{"name":"a"}{"name":"b"}`))
	var body struct {
		Name string `json:"name"`
	}

	if err := DecodeJSON(req, &body); err == nil {
		t.Fatal("expected trailing data error")
	}
}

func TestWriteError(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	WriteError(rec, http.StatusUnauthorized, "UNAUTHORIZED", "valid api key required")

	if got, want := rec.Code, http.StatusUnauthorized; got != want {
		t.Fatalf("unexpected status: got=%d want=%d", got, want)
	}
	if got, want := rec.Body.String(), "{\"error\":{\"code\":\"UNAUTHORIZED\",\"message\":\"valid api key required\"}}\n"; got != want {
		t.Fatalf("unexpected body: %q", got)
	}
}

// TestWriteJSONTurnsAnUnencodablePayloadIntoTheErrorItIs prueba el arreglo: serializar ANTES
// de escribir la cabecera.
//
// Al revés —WriteHeader y después Encode— un payload que no serializa deja el status ya
// emitido y el cuerpo truncado: el cliente recibe un 200 con JSON inválido, y la cabecera no
// se puede reescribir. Es la peor forma de fallar, porque el status miente.
func TestWriteJSONTurnsAnUnencodablePayloadIntoTheErrorItIs(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	// Un canal no tiene representación JSON.
	WriteJSON(rec, http.StatusOK, map[string]any{"broken": make(chan int)})

	if got, want := rec.Code, http.StatusInternalServerError; got != want {
		t.Fatalf("un payload que no serializa tiene que dar %d, dio %d", want, got)
	}
	if !json.Valid(rec.Body.Bytes()) {
		t.Fatalf("el cuerpo del fallback no es JSON válido: %q", rec.Body.String())
	}
	var envelope struct {
		Error APIError `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if envelope.Error.Code != "INTERNAL" {
		t.Fatalf("unexpected code %q", envelope.Error.Code)
	}
}

// TestWriteJSONKeepsTheTrailingNewline: el salto lo agregaba `Encode` y todo consumidor de
// platform ya lo recibe. Sacarlo de paso convertiría un arreglo de status en un cambio del
// cuerpo de cada respuesta del ecosistema.
func TestWriteJSONKeepsTheTrailingNewline(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	WriteJSON(rec, http.StatusOK, map[string]string{"ok": "yes"})

	if got := rec.Body.String(); !strings.HasSuffix(got, "\n") {
		t.Fatalf("el cuerpo perdió el salto final: %q", got)
	}
}
