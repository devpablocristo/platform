package admin

import (
	"context"
	"fmt"
	"strings"
	"time"

	admindomain "github.com/devpablocristo/platform/kernels/saas/go/admin/usecases/domain"
	"github.com/devpablocristo/platform/kernels/saas/go/notifications"
)

const (
	ScopeConsoleRead  = "admin:console:read"
	ScopeConsoleWrite = "admin:console:write"
)

type UseCases struct {
	repo          Repository
	notifications notifications.NotificationPort
	now           func() time.Time
}

func NewUseCases(repo Repository, notif notifications.NotificationPort) *UseCases {
	return &UseCases{
		repo:          repo,
		notifications: notif,
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

func Capabilities(role *string, scopes []string) (bool, bool) {
	canRead := hasScope(scopes, ScopeConsoleRead) || isRole(role, "owner", "admin")
	canWrite := hasScope(scopes, ScopeConsoleWrite) || isRole(role, "owner", "admin")
	return canRead, canWrite
}

func (u *UseCases) GetOrgSettings(ctx context.Context, orgID string, actor, role *string, scopes []string) (admindomain.OrgSettings, error) {
	canRead, _ := Capabilities(role, scopes)
	if !canRead {
		return admindomain.OrgSettings{}, fmt.Errorf("admin console read permission required")
	}
	item, ok, err := u.repo.GetTenantSettings(ctx, strings.TrimSpace(orgID))
	if err != nil {
		return admindomain.OrgSettings{}, err
	}
	if ok {
		return item, nil
	}
	now := u.now()
	orgID = strings.TrimSpace(orgID)
	return admindomain.OrgSettings{
		OrgID:      orgID,
		TenantID:   orgID,
		PlanCode:   "starter",
		Status:     admindomain.OrgStatusActive,
		CreatedAt:  now,
		UpdatedAt:  now,
		HardLimits: map[string]any{"tools_max": 10, "run_rpm": 30, "audit_retention_days": 30},
	}, nil
}

func (u *UseCases) UpsertOrgSettings(ctx context.Context, orgID string, actor, role *string, scopes []string, planCode string, hardLimits map[string]any) (admindomain.OrgSettings, error) {
	_, canWrite := Capabilities(role, scopes)
	if !canWrite {
		return admindomain.OrgSettings{}, fmt.Errorf("admin console write permission required")
	}
	current, err := u.GetOrgSettings(ctx, orgID, actor, role, scopes)
	if err != nil {
		return admindomain.OrgSettings{}, err
	}
	current.PlanCode = strings.TrimSpace(planCode)
	current.HardLimits = cloneMap(hardLimits)
	current.UpdatedBy = actor
	current.UpdatedAt = u.now()
	stored, err := u.repo.UpsertTenantSettings(ctx, current)
	if err != nil {
		return admindomain.OrgSettings{}, err
	}
	_ = u.repo.CreateAdminActivityEvent(ctx, admindomain.AdminActivityEvent{
		OrgID:        stored.EffectiveOrgID(),
		Actor:        actor,
		Action:       "org_settings.upsert",
		ResourceType: "org_settings",
		Payload:      map[string]any{"plan_code": stored.PlanCode},
		CreatedAt:    u.now(),
	})
	return stored, nil
}

func (u *UseCases) GetTenantSettings(ctx context.Context, tenantID string, actor, role *string, scopes []string) (admindomain.TenantSettings, error) {
	return u.GetOrgSettings(ctx, tenantID, actor, role, scopes)
}

func (u *UseCases) UpsertTenantSettings(ctx context.Context, tenantID string, actor, role *string, scopes []string, planCode string, hardLimits map[string]any) (admindomain.TenantSettings, error) {
	return u.UpsertOrgSettings(ctx, tenantID, actor, role, scopes, planCode, hardLimits)
}

func (u *UseCases) UpdateLifecycle(ctx context.Context, orgID string, actor, role *string, scopes []string, status admindomain.TenantStatus) (admindomain.TenantSettings, error) {
	_, canWrite := Capabilities(role, scopes)
	if !canWrite {
		return admindomain.TenantSettings{}, fmt.Errorf("admin console write permission required")
	}
	var deletedAt *time.Time
	if status == admindomain.TenantStatusDeleted {
		value := u.now()
		deletedAt = &value
	}
	stored, err := u.repo.UpdateTenantLifecycle(ctx, strings.TrimSpace(orgID), status, deletedAt, actor)
	if err != nil {
		return admindomain.TenantSettings{}, err
	}
	if u.notifications != nil {
		_ = u.notifications.Notify(ctx, stored.EffectiveOrgID(), "org_lifecycle_changed", map[string]string{
			"status": string(status),
		})
	}
	return stored, nil
}

func (u *UseCases) AutoSuspend(ctx context.Context, tenantID string) error {
	_, err := u.repo.UpdateTenantLifecycle(ctx, strings.TrimSpace(tenantID), admindomain.TenantStatusSuspended, nil, nil)
	return err
}

func hasScope(scopes []string, expected string) bool {
	expected = strings.TrimSpace(expected)
	for _, item := range scopes {
		if strings.TrimSpace(item) == expected {
			return true
		}
	}
	return false
}

func isRole(role *string, values ...string) bool {
	if role == nil {
		return false
	}
	current := strings.TrimSpace(strings.ToLower(*role))
	for _, item := range values {
		if current == strings.TrimSpace(strings.ToLower(item)) {
			return true
		}
	}
	return false
}

func cloneMap(values map[string]any) map[string]any {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]any, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}
