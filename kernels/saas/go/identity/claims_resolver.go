package identity

import (
	"context"
	"fmt"
	"strings"

	"github.com/devpablocristo/core/errors/go/domainerr"
	identitydomain "github.com/devpablocristo/core/saas/go/identity/usecases/domain"
)

// OrgResolverPort resuelve tenant IDs cuando el claim trae un external ID.
type OrgResolverPort interface {
	FindTenantIDByExternalID(context.Context, string) (string, bool, error)
}

// TokenVerifierPort define el adapter que valida JWT/JWKS/OIDC.
type TokenVerifierPort interface {
	VerifyToken(context.Context, string) (map[string]any, error)
}

// ClaimsConfig define el mapeo de claims del token hacia el principal.
type ClaimsConfig struct {
	Issuer      string
	Audience    string
	TenantClaim string
	RoleClaim   string
	ScopesClaim string
	ActorClaim  string
}

// ClaimsResolver resuelve un principal SaaS a partir de claims verificados.
type ClaimsResolver struct {
	verifier TokenVerifierPort
	orgs     OrgResolverPort
	cfg      ClaimsConfig
}

func NewClaimsResolver(verifier TokenVerifierPort, cfg ClaimsConfig) *ClaimsResolver {
	return NewClaimsResolverWithOrgResolver(verifier, nil, cfg)
}

func NewClaimsResolverWithOrgResolver(verifier TokenVerifierPort, orgs OrgResolverPort, cfg ClaimsConfig) *ClaimsResolver {
	return &ClaimsResolver{
		verifier: verifier,
		orgs:     orgs,
		cfg:      cfg,
	}
}

func (r *ClaimsResolver) Verify(ctx context.Context, bearerToken string) (identitydomain.Principal, error) {
	return r.ResolvePrincipal(ctx, bearerToken)
}

func (r *ClaimsResolver) ResolvePrincipal(ctx context.Context, bearerToken string) (identitydomain.Principal, error) {
	claims, err := r.verifier.VerifyToken(ctx, bearerToken)
	if err != nil {
		return identitydomain.Principal{}, domainerr.Unauthorized("invalid bearer token")
	}

	if r.cfg.Issuer != "" && normalizeIssuerURL(toString(claims["iss"])) != normalizeIssuerURL(r.cfg.Issuer) {
		return identitydomain.Principal{}, domainerr.Unauthorized("invalid token issuer")
	}
	if r.cfg.Audience != "" && !audienceMatches(claims["aud"], r.cfg.Audience) {
		return identitydomain.Principal{}, domainerr.Unauthorized("invalid token audience")
	}

	tenantID, err := r.resolveTenantID(ctx, claims)
	if err != nil {
		return identitydomain.Principal{}, err
	}

	actorClaim := r.cfg.ActorClaim
	if actorClaim == "" {
		actorClaim = "sub"
	}

	return identitydomain.Principal{
		TenantID:   tenantID,
		Actor:      strings.TrimSpace(toString(claimValue(claims, actorClaim))),
		Role:       normalizeRoleValue(firstStringClaim(claims, r.cfg.RoleClaim, "role", "org_role", "o.rol")),
		Scopes:     firstScopesClaim(claims, r.cfg.ScopesClaim, "scopes", "org_permissions", "o.per"),
		AuthMethod: "jwt",
	}, nil
}

func (r *ClaimsResolver) resolveTenantID(ctx context.Context, claims map[string]any) (string, error) {
	tenantClaim := firstStringClaim(claims, r.cfg.TenantClaim, "tenant_id", "org_id", "o.id")
	if tenantClaim == "" {
		return "", domainerr.Newf(domainerr.KindUnauthorized, "missing %s claim", effectiveTenantClaimName(r.cfg.TenantClaim))
	}
	if r.orgs == nil {
		return strings.TrimSpace(tenantClaim), nil
	}
	resolvedID, ok, err := r.orgs.FindTenantIDByExternalID(ctx, tenantClaim)
	if err != nil {
		return "", domainerr.Internal("failed resolving tenant context")
	}
	if ok && strings.TrimSpace(resolvedID) != "" {
		return strings.TrimSpace(resolvedID), nil
	}
	return "", domainerr.Unauthorized("unknown organization claim")
}

// normalizeIssuerURL compara issuers de OIDC/Clerk aunque uno lleve barra final y el otro no.
func normalizeIssuerURL(raw string) string {
	raw = strings.TrimSpace(raw)
	return strings.TrimSuffix(raw, "/")
}

func effectiveTenantClaimName(configured string) string {
	configured = strings.TrimSpace(configured)
	if configured == "" {
		return "tenant_id"
	}
	return configured
}

func firstStringClaim(claims map[string]any, names ...string) string {
	for _, name := range names {
		if value := strings.TrimSpace(toString(claimValue(claims, name))); value != "" {
			return value
		}
	}
	return ""
}

func firstScopesClaim(claims map[string]any, names ...string) []string {
	for _, name := range names {
		if value := claimValue(claims, name); value != nil {
			if scopes := parseScopes(value); len(scopes) > 0 {
				return scopes
			}
		}
	}
	return nil
}

func claimValue(claims map[string]any, path string) any {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}
	if value, ok := claims[path]; ok {
		return value
	}
	parts := strings.Split(path, ".")
	var current any = claims
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil
		}
		node, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		current, ok = node[part]
		if !ok {
			return nil
		}
	}
	return current
}

func toString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	default:
		return ""
	}
}

func audienceMatches(value any, audience string) bool {
	switch typed := value.(type) {
	case string:
		return typed == audience
	case []any:
		for _, item := range typed {
			if toString(item) == audience {
				return true
			}
		}
	case []string:
		for _, item := range typed {
			if item == audience {
				return true
			}
		}
	}
	return false
}

func parseScopes(value any) []string {
	switch typed := value.(type) {
	case string:
		return splitScopes(typed)
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			out = append(out, splitScopes(toString(item))...)
		}
		return out
	case []string:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			out = append(out, splitScopes(item)...)
		}
		return out
	default:
		return nil
	}
}

func splitScopes(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	raw = strings.ReplaceAll(raw, ",", " ")
	parts := strings.Fields(raw)
	if len(parts) == 0 {
		return nil
	}
	return parts
}

func normalizeRoleValue(role string) string {
	role = strings.TrimSpace(role)
	role = strings.TrimPrefix(role, "org:")
	return strings.TrimSpace(role)
}
