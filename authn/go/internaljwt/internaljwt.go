// Package internaljwt provee un authenticator HS256 self-contained para
// JWTs emitidos por servicios de la misma plataforma (intra-platform calls).
//
// Diseño:
//   - HS256 only. Para JWTs OIDC firmados por IdPs externos usar `authn/go/oidc`.
//   - El secret es shared entre emisor y verificador (env `*_INTERNAL_JWT_SECRET`).
//   - Issuer/audience opcionales: si se setean, son enforced.
//   - Claims aceptados (con fallback): `org_id|tenant_id|orgId`,
//     `actor_id|email|preferred_username|username|sub`, `role`, `scope|scp`.
//   - `NewAuthenticator(cfg).` retorna nil si `Secret == ""` — el caller usa
//     `NewFallback(internal, oidc)` para chainear.
//
// Lift verbatim del código duplicado en axis/companion/wire/internal_jwt.go
// y axis/nexus/wire/internal_jwt.go (ahora deletables).
package internaljwt

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	authn "github.com/devpablocristo/platform/authn/go"
)

// Config describe los parámetros del verificador interno.
type Config struct {
	Secret   string // HMAC secret compartido emisor↔verificador
	Issuer   string // opcional; si está set, se enforce
	Audience string // opcional; si está set, se enforce contra `aud` y `azp`
}

// AuthMethod es el valor de `Principal.AuthMethod` que retorna este verifier.
const AuthMethod = "internal_jwt"

// NewAuthenticator construye un authenticator que solo acepta JWTs HS256
// firmados con `cfg.Secret`. Retorna nil si Secret está vacío (caller debe
// fallar back a otro authenticator).
func NewAuthenticator(cfg Config) authn.Authenticator {
	cfg.Secret = strings.TrimSpace(cfg.Secret)
	if cfg.Secret == "" {
		return nil
	}
	expectedIssuer := strings.TrimRight(strings.TrimSpace(cfg.Issuer), "/")
	expectedAudience := strings.TrimSpace(cfg.Audience)
	return &authn.BearerJWTAuthenticator{
		Verify: hmacVerifier{secret: cfg.Secret},
		Map: func(_ context.Context, claims map[string]any) (authn.Principal, error) {
			if expectedIssuer != "" && normalizeIssuer(claims["iss"]) != expectedIssuer {
				return authn.Principal{}, errors.New("internaljwt: invalid issuer")
			}
			if expectedAudience != "" &&
				!claimContainsAudience(claims["aud"], expectedAudience) &&
				!claimContainsAudience(claims["azp"], expectedAudience) {
				return authn.Principal{}, errors.New("internaljwt: invalid audience")
			}
			sub := firstNonEmptyClaim(claims, "sub", "actor_id")
			if sub == "" {
				return authn.Principal{}, errors.New("internaljwt: missing subject")
			}
			return authn.Principal{
				OrgID:      firstNonEmptyClaim(claims, "org_id", "tenant_id", "orgId"),
				Actor:      firstNonEmptyClaim(claims, "actor_id", "email", "preferred_username", "username", "sub"),
				Role:       firstNonEmptyClaim(claims, "role"),
				Scopes:     claimScopes(claims),
				Claims:     claims,
				AuthMethod: AuthMethod,
			}, nil
		},
	}
}

// NewFallback retorna un Authenticator que prueba cada uno en orden hasta
// que alguno produzca un Principal. Tolera nils. Útil para chainear
// `internal JWT → OIDC` sin el boilerplate de un fallback type local.
func NewFallback(authns ...authn.Authenticator) authn.Authenticator {
	active := make([]authn.Authenticator, 0, len(authns))
	for _, a := range authns {
		if a != nil {
			active = append(active, a)
		}
	}
	if len(active) == 0 {
		return nil
	}
	if len(active) == 1 {
		return active[0]
	}
	return fallback{authenticators: active}
}

type fallback struct {
	authenticators []authn.Authenticator
}

func (f fallback) Authenticate(ctx context.Context, cred authn.Credential) (*authn.Principal, error) {
	var lastErr error
	for _, a := range f.authenticators {
		principal, err := a.Authenticate(ctx, cred)
		if err == nil && principal != nil {
			return principal, nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, errors.New("internaljwt: no authenticator succeeded")
}

// --- HS256 verifier (lift de companion wire/internal_jwt.go::hmacJWTVerifier) ---

type hmacVerifier struct {
	secret string
}

func (v hmacVerifier) VerifyToken(_ context.Context, rawToken string) (map[string]any, error) {
	parts := strings.Split(rawToken, ".")
	if len(parts) != 3 {
		return nil, errors.New("internaljwt: invalid token format")
	}
	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("internaljwt: decode header: %w", err)
	}
	var header struct {
		Alg string `json:"alg"`
	}
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return nil, fmt.Errorf("internaljwt: parse header: %w", err)
	}
	if header.Alg != "HS256" {
		return nil, errors.New("internaljwt: unsupported alg (HS256 only)")
	}
	unsigned := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, []byte(v.secret))
	_, _ = mac.Write([]byte(unsigned))
	expected := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(parts[2])) {
		return nil, errors.New("internaljwt: invalid signature")
	}
	claimsJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("internaljwt: decode claims: %w", err)
	}
	var claims map[string]any
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return nil, fmt.Errorf("internaljwt: parse claims: %w", err)
	}
	now := time.Now().Unix()
	if exp, ok := numericClaim(claims["exp"]); ok && exp < now {
		return nil, errors.New("internaljwt: token expired")
	}
	if nbf, ok := numericClaim(claims["nbf"]); ok && nbf > now {
		return nil, errors.New("internaljwt: token not active")
	}
	return claims, nil
}

// --- claim helpers (lift de companion wire/auth.go) ---

func normalizeIssuer(value any) string {
	return strings.TrimRight(strings.TrimSpace(claimString(value)), "/")
}

func claimString(value any) string {
	if s, ok := value.(string); ok {
		return s
	}
	return ""
}

func firstNonEmptyClaim(claims map[string]any, names ...string) string {
	for _, name := range names {
		if value := strings.TrimSpace(claimString(claims[name])); value != "" {
			return value
		}
	}
	return ""
}

func claimContainsAudience(value any, expected string) bool {
	expected = strings.TrimSpace(expected)
	if expected == "" {
		return true
	}
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v) == expected
	case []string:
		for _, item := range v {
			if strings.TrimSpace(item) == expected {
				return true
			}
		}
	case []any:
		for _, item := range v {
			if strings.TrimSpace(claimString(item)) == expected {
				return true
			}
		}
	}
	return false
}

func claimScopes(claims map[string]any) []string {
	raw := claims["scope"]
	if raw == nil {
		raw = claims["scp"]
	}
	switch v := raw.(type) {
	case string:
		parts := strings.Fields(v)
		return append([]string(nil), parts...)
	case []string:
		return append([]string(nil), v...)
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if scope := strings.TrimSpace(claimString(item)); scope != "" {
				out = append(out, scope)
			}
		}
		return out
	default:
		return nil
	}
}

func numericClaim(value any) (int64, bool) {
	switch v := value.(type) {
	case float64:
		return int64(v), true
	case int64:
		return v, true
	case json.Number:
		n, err := v.Int64()
		return n, err == nil
	default:
		return 0, false
	}
}
