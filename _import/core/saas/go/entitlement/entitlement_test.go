package entitlement

import (
	"testing"

	kerneldomain "github.com/devpablocristo/core/saas/go/kernel/usecases/domain"
)

func TestDefaultHardLimits(t *testing.T) {
	t.Parallel()

	got := DefaultHardLimits(kerneldomain.PlanEnterprise)
	if got.ToolsMax != 250 || got.RunRPM != 600 {
		t.Fatalf("unexpected enterprise limits: %#v", got)
	}
}

func TestCanUse(t *testing.T) {
	t.Parallel()

	if !CanUse(9, 10) {
		t.Fatal("expected capacity available")
	}
	if CanUse(10, 10) {
		t.Fatal("did not expect capacity available")
	}
}
