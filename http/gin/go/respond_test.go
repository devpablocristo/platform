package ginmw

import (
	"errors"
	"testing"

	"github.com/devpablocristo/core/errors/go/domainerr"
)

func TestSentinelErrorsMatchDomainKinds(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		err      error
		target   error
		wantSame bool
	}{
		{name: "not found", err: ErrNotFound, target: domainerr.NotFound(""), wantSame: true},
		{name: "conflict", err: ErrConflict, target: domainerr.Conflict(""), wantSame: true},
		{name: "forbidden", err: ErrForbidden, target: domainerr.Forbidden(""), wantSame: true},
		{name: "bad input", err: ErrBadInput, target: domainerr.Validation(""), wantSame: true},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := errors.Is(tc.err, tc.target); got != tc.wantSame {
				t.Fatalf("errors.Is(%v, %v) = %v, want %v", tc.err, tc.target, got, tc.wantSame)
			}
		})
	}
}
