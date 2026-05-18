// Package archive provides the soft-delete primitives that historically lived
// in github.com/devpablocristo/modules/crud/archive/go and were re-published
// as github.com/devpablocristo/platform/features/crud/archive/go during Ola A.
//
// In the platform monorepo, the canonical home for these helpers is
// github.com/devpablocristo/platform/lifecycle/go/archive. The API is byte-
// for-byte identical to the legacy package so consumers can migrate the
// import path and nothing else.
//
// For richer lifecycle features (BulkArchive, policies, audit hash-chain,
// retention) see the parent package github.com/devpablocristo/platform/lifecycle/go/lifecycle.
package archive

import (
	"fmt"
	"time"

	"github.com/devpablocristo/platform/errors/go/domainerr"
)

// ErrArchived indicates the operation cannot proceed because the resource is
// archived (soft-deleted). Wrapped errors satisfy domainerr.IsConflict, so
// HTTP middleware maps them to 409.
var ErrArchived = domainerr.Conflict("resource is archived")

// IfArchived returns a wrapped ErrArchived when archivedAt is non-nil.
// resource is included in the message for traceability. The pointer follows
// the GORM/SQL convention: nil = active, value = archived at that timestamp.
//
//	if err := archive.IfArchived(current.ArchivedAt, "widget"); err != nil {
//	    return err
//	}
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
