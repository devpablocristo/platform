package authn

import (
	"context"
	"errors"
	"strings"
)

// TokenClaimsVerifier verifica firma/expiración del JWT y devuelve claims crudos (sin mapeo de negocio).
// Implementaciones típicas: jwks.Verifier, oidc.DiscoveryClient.VerifyToken.
type TokenClaimsVerifier interface {
	VerifyToken(ctx context.Context, rawToken string) (map[string]any, error)
}

// MapClaimsFunc mapea claims verificados a Principal (tenant, actor, scopes, etc.).
type MapClaimsFunc func(ctx context.Context, claims map[string]any) (Principal, error)

// BearerJWTAuthenticator implementa Authenticator solo para BearerCredential:
// verifica token y aplica Map; si Map es nil, usa principal mínimo (sub + org_id/tenant_id si existen).
type BearerJWTAuthenticator struct {
	Verify TokenClaimsVerifier
	Map    MapClaimsFunc
}

// Authenticate implementa Authenticator.
func (a *BearerJWTAuthenticator) Authenticate(ctx context.Context, cred Credential) (*Principal, error) {
	if a == nil || a.Verify == nil {
		return nil, errors.New("authn: nil jwt authenticator or verifier")
	}
	if cred == nil || cred.Kind() != KindBearer {
		return nil, ErrWrongCredentialKind
	}
	bc, ok := cred.(BearerCredential)
	if !ok {
		return nil, ErrWrongCredentialKind
	}
	claims, err := a.Verify.VerifyToken(ctx, bc.Token)
	if err != nil {
		return nil, err
	}
	if a.Map != nil {
		p, merr := a.Map(ctx, claims)
		if merr != nil {
			return nil, merr
		}
		return &p, nil
	}
	return minimalPrincipalFromClaims(claims)
}

func minimalPrincipalFromClaims(claims map[string]any) (*Principal, error) {
	sub := strings.TrimSpace(toStringClaim(claims["sub"]))
	if sub == "" {
		return nil, errors.New("authn: missing sub claim")
	}
	org := firstStringClaim(claims, "org_id", "tenant_id", "orgId")
	return &Principal{
		OrgID:      org,
		Actor:      sub,
		Claims:     claims,
		AuthMethod: "jwt",
	}, nil
}

func toStringClaim(v any) string {
	switch t := v.(type) {
	case string:
		return t
	default:
		return ""
	}
}

func firstStringClaim(claims map[string]any, names ...string) string {
	for _, name := range names {
		if s := strings.TrimSpace(toStringClaim(claims[name])); s != "" {
			return s
		}
	}
	return ""
}
