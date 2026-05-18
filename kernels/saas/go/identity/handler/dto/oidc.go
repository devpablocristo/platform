package dto

type OIDCConfigResponse struct {
	OIDCEnabled bool     `json:"oidc_enabled"`
	IssuerURL   string   `json:"issuer_url,omitempty"`
	Scopes      []string `json:"scopes,omitempty"`
}

type OIDCCallbackResponse struct {
	AuthMethod  string   `json:"auth_method"`
	IDToken     string   `json:"id_token"`
	AccessToken string   `json:"access_token"`
	ExpiresIn   int      `json:"expires_in"`
	TenantID    string   `json:"tenant_id,omitempty"`
	Actor       string   `json:"actor,omitempty"`
	Role        string   `json:"role,omitempty"`
	Scopes      []string `json:"scopes,omitempty"`
}

type OIDCCallbackWarningResponse struct {
	AuthMethod  string         `json:"auth_method"`
	IDToken     string         `json:"id_token"`
	AccessToken string         `json:"access_token"`
	ExpiresIn   int            `json:"expires_in"`
	Claims      map[string]any `json:"claims"`
	Warning     string         `json:"warning"`
}
