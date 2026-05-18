package billing

import (
	"context"
	"testing"

	billingdomain "github.com/devpablocristo/core/saas/go/billing/usecases/domain"
)

func TestGetBillingStatusCreatesDefaultTenantBilling(t *testing.T) {
	t.Parallel()

	repo := &billingRepo{}
	usecases := NewUseCases(repo, billingProvider{}, nil)

	view, err := usecases.GetBillingStatus(context.Background(), "acme")
	if err != nil {
		t.Fatalf("GetBillingStatus returned error: %v", err)
	}
	if view.PlanCode != billingdomain.PlanStarter {
		t.Fatalf("unexpected plan: %s", view.PlanCode)
	}
	if view.HardLimits.ToolsMax != 10 {
		t.Fatalf("unexpected limits: %#v", view.HardLimits)
	}
}

func TestApplyPlanChange(t *testing.T) {
	t.Parallel()

	repo := &billingRepo{}
	usecases := NewUseCases(repo, billingProvider{}, nil)
	item, err := usecases.ApplyPlanChange(context.Background(), "acme", billingdomain.PlanEnterprise, billingdomain.BillingActive, nil)
	if err != nil {
		t.Fatalf("ApplyPlanChange returned error: %v", err)
	}
	if item.PlanCode != billingdomain.PlanEnterprise {
		t.Fatalf("unexpected plan: %s", item.PlanCode)
	}
	if item.BillingStatus != billingdomain.BillingActive {
		t.Fatalf("unexpected status: %s", item.BillingStatus)
	}
}

type billingRepo struct {
	item billingdomain.TenantBilling
}

func (r *billingRepo) GetTenantBilling(context.Context, string) (billingdomain.TenantBilling, bool, error) {
	if r.item.TenantID == "" {
		return billingdomain.TenantBilling{}, false, nil
	}
	return r.item, true, nil
}
func (r *billingRepo) UpsertTenantBilling(_ context.Context, item billingdomain.TenantBilling) (billingdomain.TenantBilling, error) {
	r.item = item
	return item, nil
}
func (r *billingRepo) GetUsageSummary(context.Context, string) (billingdomain.UsageSummary, error) {
	return billingdomain.UsageSummary{Period: "2026-03"}, nil
}

type billingProvider struct{}

func (billingProvider) CreateCheckoutSession(context.Context, billingdomain.CheckoutInput, billingdomain.TenantBilling) (string, error) {
	return "https://checkout.local/session", nil
}
func (billingProvider) CreatePortalSession(context.Context, billingdomain.PortalInput, billingdomain.TenantBilling) (string, error) {
	return "https://portal.local/session", nil
}
