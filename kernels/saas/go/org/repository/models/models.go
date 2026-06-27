package models

import (
	"time"

	orgdomain "github.com/devpablocristo/platform/kernels/saas/go/org/usecases/domain"
)

type Organization struct {
	ID         string    `json:"id"`
	ExternalID string    `json:"external_id,omitempty"`
	Name       string    `json:"name"`
	CreatedAt  time.Time `json:"created_at"`
}

type APIKey struct {
	ID         string    `json:"id"`
	OrgID      string    `json:"org_id"`
	APIKeyHash string    `json:"api_key_hash"`
	Name       string    `json:"name"`
	Scopes     []string  `json:"scopes,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

type Principal struct {
	OrgID    string   `json:"org_id"`
	TenantID string   `json:"-"`
	Scopes   []string `json:"scopes,omitempty"`
}

func PrincipalFromDomain(item orgdomain.Principal) Principal {
	return Principal{
		OrgID:    item.EffectiveOrgID(),
		TenantID: item.TenantID,
		Scopes:   append([]string(nil), item.Scopes...),
	}
}

func (m Principal) ToDomain() orgdomain.Principal {
	return orgdomain.Principal{
		OrgID:    m.OrgID,
		TenantID: m.TenantID,
		Scopes:   append([]string(nil), m.Scopes...),
	}
}
