package fsm

import (
	"errors"
	"fmt"

	"github.com/devpablocristo/platform/errors/go/domainerr"
)

// MapDomainError translates the FSM sentinel errors (ErrTerminal,
// ErrInvalidTransition) into domainerr.Conflict so that HTTP middleware
// (e.g. platform/http/gin/go.Respond) returns 409 with a readable message.
// Unrecognized errors are wrapped with %w so callers can keep using
// errors.Is on their own sentinels.
//
// `current` and `next` are passed as plain strings — this helper does not
// know any product status vocabulary (§ Invariante I1).
//
// Typical usage from a domain usecase:
//
//	if err := stateMachine.Validate(current.Status, next); err != nil {
//	    return entity{}, fsm.MapDomainError(current.Status, next, err)
//	}
func MapDomainError(current, next string, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrTerminal) {
		return domainerr.Newf(domainerr.KindConflict, "status %q is terminal", current)
	}
	if errors.Is(err, ErrInvalidTransition) {
		return domainerr.Newf(domainerr.KindConflict, "status transition not allowed: %s -> %s", current, next)
	}
	return fmt.Errorf("fsm validation: %w", err)
}
