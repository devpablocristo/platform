package models

import identitydomain "github.com/devpablocristo/core/saas/go/identity/usecases/domain"

type Principal struct {
	TenantID   string   `json:"tenant_id"`
	Actor      string   `json:"actor,omitempty"`
	Role       string   `json:"role,omitempty"`
	Scopes     []string `json:"scopes,omitempty"`
	AuthMethod string   `json:"auth_method,omitempty"`
}

func FromDomain(item identitydomain.Principal) Principal {
	return Principal{
		TenantID:   item.TenantID,
		Actor:      item.Actor,
		Role:       item.Role,
		Scopes:     append([]string(nil), item.Scopes...),
		AuthMethod: item.AuthMethod,
	}
}

func (m Principal) ToDomain() identitydomain.Principal {
	return identitydomain.Principal{
		TenantID:   m.TenantID,
		Actor:      m.Actor,
		Role:       m.Role,
		Scopes:     append([]string(nil), m.Scopes...),
		AuthMethod: m.AuthMethod,
	}
}
