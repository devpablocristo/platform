package httperr

import (
	"bytes"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/devpablocristo/platform/errors/go/domainerr"
)

func TestNormalizeDomainerrUsesMessage(t *testing.T) {
	t.Parallel()
	err := domainerr.NotFound("party  missing")
	st, api := Normalize(err)
	if st != http.StatusNotFound {
		t.Fatalf("status: %d", st)
	}
	if api.Message != "party  missing" {
		t.Fatalf("message: %q", api.Message)
	}
}

func TestNormalizeDomainerrBusinessRule(t *testing.T) {
	t.Parallel()
	err := domainerr.BusinessRule("credit note is not active")
	st, api := Normalize(err)
	if st != http.StatusUnprocessableEntity {
		t.Fatalf("status: %d", st)
	}
	if api.Code != "BUSINESS_RULE" {
		t.Fatalf("code: %q", api.Code)
	}
}

func TestNormalizeDomainerrUnavailable(t *testing.T) {
	t.Parallel()
	err := domainerr.Unavailable("service down")
	st, _ := Normalize(err)
	if st != http.StatusServiceUnavailable {
		t.Fatalf("status: %d", st)
	}
}

func TestNormalizeDomainerrUpstreamError(t *testing.T) {
	t.Parallel()
	err := domainerr.UpstreamError("bad gateway")
	st, _ := Normalize(err)
	if st != http.StatusBadGateway {
		t.Fatalf("status: %d", st)
	}
}

func TestNormalizeHTTPError(t *testing.T) {
	t.Parallel()
	err := New(http.StatusBadGateway, "UPSTREAM", "bad")
	st, api := Normalize(err)
	if st != http.StatusBadGateway || api.Code != "UPSTREAM" {
		t.Fatalf("got %d %+v", st, api)
	}
}

func TestNormalizeUnknown(t *testing.T) {
	t.Parallel()
	st, api := Normalize(errors.New("secret"))
	if st != http.StatusInternalServerError || api.Code != CodeInternal {
		t.Fatalf("got %d %+v", st, api)
	}
}

// TestWriteFromLogsTheInternalErrorItHides prueba el segundo arreglo.
//
// Un error que no es de dominio sale como `{"code":"INTERNAL","message":"internal error"}`.
// Sin log, ese 500 no se puede debuggear: el error real desaparece. Y al revés, un error
// clasificado NO se loguea —un 404 es una respuesta esperada del negocio, no un incidente—
// porque si ensuciara el log, el log dejaría de servir para encontrar incidentes.
func TestWriteFromLogsTheInternalErrorItHides(t *testing.T) {
	previous := slog.Default()
	defer slog.SetDefault(previous)

	var logged bytes.Buffer
	slog.SetDefault(slog.New(slog.NewTextHandler(&logged, &slog.HandlerOptions{Level: slog.LevelError})))

	rec := httptest.NewRecorder()
	WriteFrom(rec, errors.New("db connection refused at 10.0.0.5"))

	if got, want := rec.Code, http.StatusInternalServerError; got != want {
		t.Fatalf("unexpected status: got=%d want=%d", got, want)
	}
	if !strings.Contains(logged.String(), "10.0.0.5") {
		t.Fatalf("el error real no quedó en el log: %q", logged.String())
	}
	// Y no se filtra al cliente.
	if strings.Contains(rec.Body.String(), "10.0.0.5") {
		t.Fatalf("el error interno se filtró en la respuesta: %q", rec.Body.String())
	}

	logged.Reset()
	rec = httptest.NewRecorder()
	WriteFrom(rec, domainerr.NotFound("the document does not exist"))

	if got, want := rec.Code, http.StatusNotFound; got != want {
		t.Fatalf("unexpected status: got=%d want=%d", got, want)
	}
	if logged.Len() != 0 {
		t.Fatalf("un error de dominio no puede loguearse como incidente: %q", logged.String())
	}
}
