package dto

import tenantdomain "github.com/devpablocristo/core/saas/go/tenant/usecases/domain"

type NormalizeRequest struct {
	Value string `json:"value"`
}

type StringResponse struct {
	Value string `json:"value"`
}

type NewMembershipRequest struct {
	TenantID string `json:"tenant_id"`
	UserID   string `json:"user_id"`
	Role     string `json:"role"`
}

type NewMembershipResponse struct {
	Membership tenantdomain.Membership `json:"membership"`
}
