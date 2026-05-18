package billing

import (
	"context"
	"fmt"
	"strings"
	"time"

	billingdomain "github.com/devpablocristo/core/saas/go/billing/usecases/domain"
	"github.com/devpablocristo/core/saas/go/notifications"
)

type UseCases struct {
	repo          Repository
	provider      CheckoutProvider
	notifications notifications.NotificationPort
	now           func() time.Time
}

func NewUseCases(repo Repository, provider CheckoutProvider, notif notifications.NotificationPort) *UseCases {
	return &UseCases{
		repo:          repo,
		provider:      provider,
		notifications: notif,
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

func DefaultHardLimits(plan billingdomain.PlanCode) billingdomain.HardLimits {
	switch normalizePlan(plan) {
	case billingdomain.PlanGrowth:
		return billingdomain.HardLimits{ToolsMax: 50, RunRPM: 120, AuditRetentionDays: 90}
	case billingdomain.PlanEnterprise:
		return billingdomain.HardLimits{ToolsMax: 250, RunRPM: 600, AuditRetentionDays: 365}
	default:
		return billingdomain.HardLimits{ToolsMax: 10, RunRPM: 30, AuditRetentionDays: 30}
	}
}

func (u *UseCases) EnsureTenantBilling(ctx context.Context, tenantID string) (billingdomain.TenantBilling, error) {
	current, ok, err := u.repo.GetTenantBilling(ctx, strings.TrimSpace(tenantID))
	if err != nil {
		return billingdomain.TenantBilling{}, err
	}
	if ok {
		return current, nil
	}
	now := u.now()
	return u.repo.UpsertTenantBilling(ctx, billingdomain.TenantBilling{
		TenantID:      strings.TrimSpace(tenantID),
		PlanCode:      billingdomain.PlanStarter,
		HardLimits:    DefaultHardLimits(billingdomain.PlanStarter),
		BillingStatus: billingdomain.BillingTrialing,
		CreatedAt:     now,
		UpdatedAt:     now,
	})
}

func (u *UseCases) GetBillingStatus(ctx context.Context, tenantID string) (billingdomain.BillingStatusView, error) {
	settings, err := u.EnsureTenantBilling(ctx, tenantID)
	if err != nil {
		return billingdomain.BillingStatusView{}, err
	}
	usage, err := u.repo.GetUsageSummary(ctx, strings.TrimSpace(tenantID))
	if err != nil {
		return billingdomain.BillingStatusView{}, err
	}
	return billingdomain.BillingStatusView{
		PlanCode:      settings.PlanCode,
		BillingStatus: settings.BillingStatus,
		HardLimits:    settings.HardLimits,
		Usage:         usage,
	}, nil
}

func (u *UseCases) CreateCheckoutSession(ctx context.Context, input billingdomain.CheckoutInput) (string, error) {
	if u.provider == nil {
		return "", fmt.Errorf("billing provider is required")
	}
	input.TenantID = strings.TrimSpace(input.TenantID)
	if input.TenantID == "" {
		return "", fmt.Errorf("tenant id is required")
	}
	input.PlanCode = normalizePlan(input.PlanCode)
	if input.PlanCode == "" {
		return "", fmt.Errorf("plan code is required")
	}
	settings, err := u.EnsureTenantBilling(ctx, input.TenantID)
	if err != nil {
		return "", err
	}
	return u.provider.CreateCheckoutSession(ctx, input, settings)
}

func (u *UseCases) CreatePortalSession(ctx context.Context, input billingdomain.PortalInput) (string, error) {
	if u.provider == nil {
		return "", fmt.Errorf("billing provider is required")
	}
	input.TenantID = strings.TrimSpace(input.TenantID)
	if input.TenantID == "" {
		return "", fmt.Errorf("tenant id is required")
	}
	settings, err := u.EnsureTenantBilling(ctx, input.TenantID)
	if err != nil {
		return "", err
	}
	return u.provider.CreatePortalSession(ctx, input, settings)
}

func (u *UseCases) ApplyPlanChange(ctx context.Context, tenantID string, plan billingdomain.PlanCode, status billingdomain.BillingStatus, actor *string) (billingdomain.TenantBilling, error) {
	current, err := u.EnsureTenantBilling(ctx, tenantID)
	if err != nil {
		return billingdomain.TenantBilling{}, err
	}
	current.PlanCode = normalizePlan(plan)
	current.HardLimits = DefaultHardLimits(current.PlanCode)
	current.BillingStatus = normalizeStatus(status)
	current.UpdatedAt = u.now()
	stored, err := u.repo.UpsertTenantBilling(ctx, current)
	if err != nil {
		return billingdomain.TenantBilling{}, err
	}
	if u.notifications != nil {
		_ = u.notifications.Notify(ctx, tenantID, "billing_plan_changed", map[string]string{
			"plan_code": string(stored.PlanCode),
			"status":    string(stored.BillingStatus),
			"actor":     deref(actor),
		})
	}
	return stored, nil
}

func normalizePlan(plan billingdomain.PlanCode) billingdomain.PlanCode {
	switch billingdomain.PlanCode(strings.ToLower(strings.TrimSpace(string(plan)))) {
	case billingdomain.PlanGrowth:
		return billingdomain.PlanGrowth
	case billingdomain.PlanEnterprise:
		return billingdomain.PlanEnterprise
	default:
		return billingdomain.PlanStarter
	}
}

func normalizeStatus(status billingdomain.BillingStatus) billingdomain.BillingStatus {
	switch billingdomain.BillingStatus(strings.ToLower(strings.TrimSpace(string(status)))) {
	case billingdomain.BillingActive:
		return billingdomain.BillingActive
	case billingdomain.BillingPastDue:
		return billingdomain.BillingPastDue
	case billingdomain.BillingCanceled:
		return billingdomain.BillingCanceled
	case billingdomain.BillingUnpaid:
		return billingdomain.BillingUnpaid
	default:
		return billingdomain.BillingTrialing
	}
}

func deref(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}
