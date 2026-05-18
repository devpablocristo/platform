package httperr

import (
	"errors"
	"net/http"
	"testing"

	"github.com/devpablocristo/core/errors/go/domainerr"
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
