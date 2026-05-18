// Package authn provides unified inbound authentication for HTTP services.
// Responsibility: extract credential → verify → produce Principal.
// NOT responsible for: sessions, refresh, OAuth flows, identity management.
package authn

import (
	"context"
	"strings"
)

// Principal es la identidad resuelta de una credencial verificada.
type Principal struct {
	OrgID      string         `json:"org_id"`
	Actor      string         `json:"actor"`
	Role       string         `json:"role"`
	Scopes     []string       `json:"scopes"`
	Claims     map[string]any `json:"claims,omitempty"`
	AuthMethod string         `json:"auth_method,omitempty"` // ej. jwt, api_key, bearer (lo define el verificador)
}

// Credential representa una credencial extraída de un request.
type Credential interface {
	Kind() string
}

// BearerCredential credencial de tipo Bearer JWT.
type BearerCredential struct {
	Token string
}

func (BearerCredential) Kind() string { return KindBearer }

// APIKeyCredential credencial de tipo API Key.
type APIKeyCredential struct {
	Key string
}

func (APIKeyCredential) Kind() string { return KindAPIKey }

// Authenticator verifica una credencial y produce un Principal.
type Authenticator interface {
	Authenticate(ctx context.Context, cred Credential) (*Principal, error)
}

// Extractor extrae una credencial de un HTTP request.
// Retorna nil, nil si no hay credencial presente (permite chaining).
type Extractor interface {
	Extract(authHeader, apiKeyHeader string) (Credential, error)
}

// --- Extractors built-in ---

// BearerToken extrae un Bearer token del header Authorization.
func BearerToken(raw string) (string, bool) {
	const prefix = "Bearer "
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(raw, prefix) {
		return "", false
	}
	token := strings.TrimSpace(strings.TrimPrefix(raw, prefix))
	return token, token != ""
}

// APIKeyToken extrae API key de X-API-Key header o Authorization "ApiKey <key>".
func APIKeyToken(authHeader, xAPIKey string) (string, bool) {
	if value := strings.TrimSpace(xAPIKey); value != "" {
		return value, true
	}
	const prefix = "ApiKey "
	authHeader = strings.TrimSpace(authHeader)
	if !strings.HasPrefix(authHeader, prefix) {
		return "", false
	}
	token := strings.TrimSpace(strings.TrimPrefix(authHeader, prefix))
	return token, token != ""
}
