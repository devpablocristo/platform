package admin

import (
	"context"
	"testing"
	"time"

	admindomain "github.com/devpablocristo/core/saas/go/admin/usecases/domain"
)

func TestCapabilities(t *testing.T) {
	t.Parallel()

	role := "viewer"
	canRead, canWrite := Capabilities(&role, []string{ScopeConsoleRead})
	if !canRead || canWrite {
		t.Fatalf("unexpected capabilities: read=%v write=%v", canRead, canWrite)
	}
}

func TestUpsertTenantSettings(t *testing.T) {
	t.Parallel()

	role := "admin"
	repo := &adminRepo{}
	usecases := NewUseCases(repo, nil)

	item, err := usecases.UpsertTenantSettings(context.Background(), "acme", nil, &role, nil, "growth", map[string]any{"tools_max": 20})
	if err != nil {
		t.Fatalf("UpsertTenantSettings returned error: %v", err)
	}
	if item.PlanCode != "growth" {
		t.Fatalf("unexpected plan code: %q", item.PlanCode)
	}
}

type adminRepo struct{}

func (r *adminRepo) GetTenantSettings(context.Context, string) (admindomain.TenantSettings, bool, error) {
	return admindomain.TenantSettings{}, false, nil
}
func (r *adminRepo) UpsertTenantSettings(_ context.Context, item admindomain.TenantSettings) (admindomain.TenantSettings, error) {
	return item, nil
}
func (r *adminRepo) UpdateTenantLifecycle(_ context.Context, tenantID string, status admindomain.TenantStatus, deletedAt *time.Time, updatedBy *string) (admindomain.TenantSettings, error) {
	return admindomain.TenantSettings{TenantID: tenantID, Status: status, DeletedAt: deletedAt, UpdatedBy: updatedBy}, nil
}
func (r *adminRepo) CreateAdminActivityEvent(context.Context, admindomain.AdminActivityEvent) error {
	return nil
}
func (r *adminRepo) ListAdminActivityEvents(context.Context, string, int) ([]admindomain.AdminActivityEvent, error) {
	return nil, nil
}
