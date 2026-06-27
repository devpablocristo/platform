package models

import identitydomain "github.com/devpablocristo/platform/kernels/saas/go/identity/usecases/domain"

type Principal struct {
	OrgID      string   `json:"org_id"`
	TenantID   string   `json:"-"`
	Actor      string   `json:"actor,omitempty"`
	Role       string   `json:"role,omitempty"`
	Scopes     []string `json:"scopes,omitempty"`
	AuthMethod string   `json:"auth_method,omitempty"`
}

func FromDomain(item identitydomain.Principal) Principal {
	return Principal{
		OrgID:      item.EffectiveOrgID(),
		TenantID:   item.TenantID,
		Actor:      item.Actor,
		Role:       item.Role,
		Scopes:     append([]string(nil), item.Scopes...),
		AuthMethod: item.AuthMethod,
	}
}

func (m Principal) ToDomain() identitydomain.Principal {
	return identitydomain.Principal{
		OrgID:      m.OrgID,
		TenantID:   m.TenantID,
		Actor:      m.Actor,
		Role:       m.Role,
		Scopes:     append([]string(nil), m.Scopes...),
		AuthMethod: m.AuthMethod,
	}
}
