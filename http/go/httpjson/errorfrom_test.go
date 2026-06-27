package httpjson

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/devpablocristo/platform/errors/go/domainerr"
)

func TestStatusForError(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		err  error
		want int
	}{
		{"not_found", domainerr.New(domainerr.KindNotFound, "x"), http.StatusNotFound},
		{"validation", domainerr.New(domainerr.KindValidation, "x"), http.StatusBadRequest},
		{"forbidden", domainerr.New(domainerr.KindForbidden, "x"), http.StatusForbidden},
		{"conflict", domainerr.New(domainerr.KindConflict, "x"), http.StatusConflict},
		{"unauthorized", domainerr.New(domainerr.KindUnauthorized, "x"), http.StatusUnauthorized},
		{"unclassified", errors.New("boom"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		if got := StatusForError(tc.err); got != tc.want {
			t.Fatalf("%s: got=%d want=%d", tc.name, got, tc.want)
		}
	}
}

func TestWriteFlatErrorFromSurfacesClientSafeMessage(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	WriteFlatErrorFrom(rec, domainerr.New(domainerr.KindValidation, "name is required"), "ctx")

	if got, want := rec.Code, http.StatusBadRequest; got != want {
		t.Fatalf("status got=%d want=%d", got, want)
	}
	if body := rec.Body.String(); !strings.Contains(body, `"VALIDATION"`) || !strings.Contains(body, "name is required") {
		t.Fatalf("unexpected body: %q", body)
	}
}

func TestWriteFlatErrorFromHidesNotFoundAndInternalDetail(t *testing.T) {
	t.Parallel()

	// NOT_FOUND uses a generic message (no leakage of the underlying error).
	rec := httptest.NewRecorder()
	WriteFlatErrorFrom(rec, domainerr.New(domainerr.KindNotFound, "secret table row 42"), "ctx")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status got=%d want=%d", rec.Code, http.StatusNotFound)
	}
	if body := rec.Body.String(); strings.Contains(body, "secret table") {
		t.Fatalf("not-found leaked detail: %q", body)
	}

	// Unclassified → 500 with a generic message.
	rec = httptest.NewRecorder()
	WriteFlatErrorFrom(rec, errors.New("db connection refused at 10.0.0.5"), "ctx")
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status got=%d want=%d", rec.Code, http.StatusInternalServerError)
	}
	if body := rec.Body.String(); strings.Contains(body, "10.0.0.5") {
		t.Fatalf("internal error leaked detail: %q", body)
	}
}
