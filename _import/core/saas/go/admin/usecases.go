package admin

import (
	"context"
	"fmt"
	"strings"
	"time"

	admindomain "github.com/devpablocristo/core/saas/go/admin/usecases/domain"
	"github.com/devpablocristo/core/saas/go/notifications"
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

func (u *UseCases) GetTenantSettings(ctx context.Context, tenantID string, actor, role *string, scopes []string) (admindomain.TenantSettings, error) {
	canRead, _ := Capabilities(role, scopes)
	if !canRead {
		return admindomain.TenantSettings{}, fmt.Errorf("admin console read permission required")
	}
	item, ok, err := u.repo.GetTenantSettings(ctx, strings.TrimSpace(tenantID))
	if err != nil {
		return admindomain.TenantSettings{}, err
	}
	if ok {
		return item, nil
	}
	now := u.now()
	return admindomain.TenantSettings{
		TenantID:   strings.TrimSpace(tenantID),
		PlanCode:   "starter",
		Status:     admindomain.TenantStatusActive,
		CreatedAt:  now,
		UpdatedAt:  now,
		HardLimits: map[string]any{"tools_max": 10, "run_rpm": 30, "audit_retention_days": 30},
	}, nil
}

func (u *UseCases) UpsertTenantSettings(ctx context.Context, tenantID string, actor, role *string, scopes []string, planCode string, hardLimits map[string]any) (admindomain.TenantSettings, error) {
	_, canWrite := Capabilities(role, scopes)
	if !canWrite {
		return admindomain.TenantSettings{}, fmt.Errorf("admin console write permission required")
	}
	current, err := u.GetTenantSettings(ctx, tenantID, actor, role, scopes)
	if err != nil {
		return admindomain.TenantSettings{}, err
	}
	current.PlanCode = strings.TrimSpace(planCode)
	current.HardLimits = cloneMap(hardLimits)
	current.UpdatedBy = actor
	current.UpdatedAt = u.now()
	stored, err := u.repo.UpsertTenantSettings(ctx, current)
	if err != nil {
		return admindomain.TenantSettings{}, err
	}
	_ = u.repo.CreateAdminActivityEvent(ctx, admindomain.AdminActivityEvent{
		TenantID:     stored.TenantID,
		Actor:        actor,
		Action:       "tenant_settings.upsert",
		ResourceType: "tenant_settings",
		Payload:      map[string]any{"plan_code": stored.PlanCode},
		CreatedAt:    u.now(),
	})
	return stored, nil
}

func (u *UseCases) UpdateLifecycle(ctx context.Context, tenantID string, actor, role *string, scopes []string, status admindomain.TenantStatus) (admindomain.TenantSettings, error) {
	_, canWrite := Capabilities(role, scopes)
	if !canWrite {
		return admindomain.TenantSettings{}, fmt.Errorf("admin console write permission required")
	}
	var deletedAt *time.Time
	if status == admindomain.TenantStatusDeleted {
		value := u.now()
		deletedAt = &value
	}
	stored, err := u.repo.UpdateTenantLifecycle(ctx, strings.TrimSpace(tenantID), status, deletedAt, actor)
	if err != nil {
		return admindomain.TenantSettings{}, err
	}
	if u.notifications != nil {
		_ = u.notifications.Notify(ctx, stored.TenantID, "tenant_lifecycle_changed", map[string]string{
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
