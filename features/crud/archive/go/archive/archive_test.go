package archive_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/devpablocristo/core/errors/go/domainerr"
	"github.com/devpablocristo/modules/crud/archive/go/archive"
)

func TestIfArchivedReturnsNilWhenActive(t *testing.T) {
	t.Parallel()
	if err := archive.IfArchived(nil, "product"); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestIfArchivedReturnsErrArchivedWhenSet(t *testing.T) {
	t.Parallel()
	now := time.Now()
	err := archive.IfArchived(&now, "product")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, archive.ErrArchived) {
		t.Fatalf("expected error wrapping ErrArchived, got %v", err)
	}
	if !strings.Contains(err.Error(), "product archived") {
		t.Fatalf("expected message to include resource, got %q", err.Error())
	}
	// ErrArchived debe ser un domainerr.Conflict para que los routers
	// HTTP lo mapeen a 409.
	var de domainerr.Error
	if !errors.As(err, &de) {
		t.Fatalf("expected error to unwrap to domainerr.Error, got %v", err)
	}
	if de.Kind() != domainerr.KindConflict {
		t.Fatalf("expected KindConflict, got %v", de.Kind())
	}
}

func TestIsArchived(t *testing.T) {
	t.Parallel()
	if archive.IsArchived(nil) {
		t.Fatal("expected false for nil")
	}
	now := time.Now()
	if !archive.IsArchived(&now) {
		t.Fatal("expected true when archivedAt is set")
	}
}
