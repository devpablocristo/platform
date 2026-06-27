package dto

import admindomain "github.com/devpablocristo/platform/kernels/saas/go/admin/usecases/domain"

type GetOrgSettingsRequest struct {
	OrgID    string   `json:"org_id"`
	TenantID string   `json:"-"`
	Actor    *string  `json:"actor,omitempty"`
	Role     *string  `json:"role,omitempty"`
	Scopes   []string `json:"scopes,omitempty"`
}

type UpsertOrgSettingsRequest struct {
	OrgID      string         `json:"org_id"`
	TenantID   string         `json:"-"`
	Actor      *string        `json:"actor,omitempty"`
	Role       *string        `json:"role,omitempty"`
	Scopes     []string       `json:"scopes,omitempty"`
	PlanCode   string         `json:"plan_code"`
	HardLimits map[string]any `json:"hard_limits,omitempty"`
}

type UpdateLifecycleRequest struct {
	OrgID    string                   `json:"org_id"`
	TenantID string                   `json:"-"`
	Actor    *string                  `json:"actor,omitempty"`
	Role     *string                  `json:"role,omitempty"`
	Scopes   []string                 `json:"scopes,omitempty"`
	Status   admindomain.TenantStatus `json:"status"`
}

type OrgSettingsResponse struct {
	Settings admindomain.OrgSettings `json:"settings"`
}

type GetTenantSettingsRequest = GetOrgSettingsRequest
type UpsertTenantSettingsRequest = UpsertOrgSettingsRequest
type TenantSettingsResponse = OrgSettingsResponse
