package dto

import admindomain "github.com/devpablocristo/core/saas/go/admin/usecases/domain"

type GetTenantSettingsRequest struct {
	TenantID string   `json:"tenant_id"`
	Actor    *string  `json:"actor,omitempty"`
	Role     *string  `json:"role,omitempty"`
	Scopes   []string `json:"scopes,omitempty"`
}

type UpsertTenantSettingsRequest struct {
	TenantID   string         `json:"tenant_id"`
	Actor      *string        `json:"actor,omitempty"`
	Role       *string        `json:"role,omitempty"`
	Scopes     []string       `json:"scopes,omitempty"`
	PlanCode   string         `json:"plan_code"`
	HardLimits map[string]any `json:"hard_limits,omitempty"`
}

type UpdateLifecycleRequest struct {
	TenantID string                   `json:"tenant_id"`
	Actor    *string                  `json:"actor,omitempty"`
	Role     *string                  `json:"role,omitempty"`
	Scopes   []string                 `json:"scopes,omitempty"`
	Status   admindomain.TenantStatus `json:"status"`
}

type TenantSettingsResponse struct {
	Settings admindomain.TenantSettings `json:"settings"`
}
