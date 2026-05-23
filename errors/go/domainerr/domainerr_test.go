package domainerr_test

import (
	"errors"
	"testing"

	"github.com/devpablocristo/platform/errors/go/domainerr"
)

func TestTenantMissing_KindForbidden(t *testing.T) {
	t.Parallel()
	err := domainerr.TenantMissing()
	if !domainerr.IsForbidden(err) {
		t.Fatalf("expected FORBIDDEN kind, got %v", err.Kind())
	}
	if err.Message() != "tenant context required" {
		t.Errorf("unexpected message: %q", err.Message())
	}
}

func TestTenantMismatch_KindForbidden(t *testing.T) {
	t.Parallel()
	err := domainerr.TenantMismatch()
	if !domainerr.IsForbidden(err) {
		t.Fatalf("expected FORBIDDEN kind, got %v", err.Kind())
	}
	if err.Message() != "tenant mismatch" {
		t.Errorf("unexpected message: %q", err.Message())
	}
}

func TestTenantNotFound_FormatsID(t *testing.T) {
	t.Parallel()
	err := domainerr.TenantNotFound("acme")
	if !domainerr.IsNotFound(err) {
		t.Fatalf("expected NOT_FOUND kind, got %v", err.Kind())
	}
	if err.Message() != "tenant 'acme' not found" {
		t.Errorf("unexpected message: %q", err.Message())
	}
}

func TestTenantNotFound_EmptyID(t *testing.T) {
	t.Parallel()
	err := domainerr.TenantNotFound("")
	if !domainerr.IsNotFound(err) {
		t.Fatalf("expected NOT_FOUND kind, got %v", err.Kind())
	}
	if err.Message() != "tenant not found" {
		t.Errorf("unexpected message: %q", err.Message())
	}
}

func TestTenantFactories_ErrorsIsByKind(t *testing.T) {
	t.Parallel()
	missing := domainerr.TenantMissing()
	mismatch := domainerr.TenantMismatch()
	if !errors.Is(missing, mismatch) {
		t.Error("FORBIDDEN-kind errors should match each other via errors.Is (kind equality)")
	}
	notFound := domainerr.TenantNotFound("x")
	if errors.Is(missing, notFound) {
		t.Error("FORBIDDEN should not match NOT_FOUND")
	}
}
