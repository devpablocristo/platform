package domain

import "time"

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

func (p Principal) EffectiveOrgID() string {
	if p.OrgID != "" {
		return p.OrgID
	}
	return p.TenantID
}
