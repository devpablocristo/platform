// Package archive provides small helpers for resources in archived state.
//
// Archive means retained but outside active workflows. It is not trash and it
// is not deletion.
package archive

import (
	"fmt"
	"time"

	"github.com/devpablocristo/platform/errors/go/domainerr"
)

// ErrArchived indicates the operation cannot proceed because the resource is
// archived. Wrapped errors satisfy domainerr.IsConflict, so HTTP middleware
// maps them to 409.
var ErrArchived = domainerr.Conflict("resource is archived")

// IfArchived returns a wrapped ErrArchived when archivedAt is non-nil.
func IfArchived(archivedAt *time.Time, resource string) error {
	if archivedAt == nil {
		return nil
	}
	return fmt.Errorf("%s archived: %w", resource, ErrArchived)
}

// IsArchived is the pure predicate equivalent of archivedAt != nil.
func IsArchived(archivedAt *time.Time) bool {
	return archivedAt != nil
}
