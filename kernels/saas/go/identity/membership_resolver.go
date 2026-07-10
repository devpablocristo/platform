package identity

import (
	"context"
	"strings"

	"github.com/devpablocristo/platform/errors/go/domainerr"
	identitydomain "github.com/devpablocristo/platform/kernels/saas/go/identity/usecases/domain"
)

// ActorTenant es una membership de un actor: el tenant donde opera + su rol y scopes. Es lo que el
// consumer devuelve desde su tabla de memberships (p.ej. auth_memberships).
type ActorTenant struct {
	TenantID string
	Role     string
	Scopes   []string
}

// MembershipResolverPort resuelve las memberships de un actor por su sub del IdP. La impl la da el
// consumer (lookup en su tabla de memberships).
type MembershipResolverPort interface {
	TenantsForActor(ctx context.Context, actorSub string) ([]ActorTenant, error)
}

type requestedTenantKey struct{}

// WithRequestedTenant estampa el tenant pedido explícitamente (p.ej. el header X-Tenant-Id) en el ctx,
// para que el MembershipResolver lo use cuando el actor tiene más de una membership.
func WithRequestedTenant(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, requestedTenantKey{}, strings.TrimSpace(tenantID))
}

// RequestedTenantFromContext lee el tenant pedido (vacío si no se estampó).
func RequestedTenantFromContext(ctx context.Context) string {
	v, _ := ctx.Value(requestedTenantKey{}).(string)
	return v
}

// MembershipConfig configura el MembershipResolver. Issuer/Audience se enforzan igual que en el
// ClaimsResolver (fail-closed si están seteados). ActorClaim default "sub".
type MembershipConfig struct {
	Issuer     string
	Audience   string
	ActorClaim string
}

// MembershipResolver verifica un ID token y deriva el Principal (tenant/role/scopes) desde las
// memberships del actor — para IdPs que NO ponen org/role/scopes en el token (p.ej. Identity
// Platform/Firebase). Aplica la política de selección de tenant 0/1/>1:
//   - tenant pedido explícito (X-Tenant-Id) → debe existir una membership en ese tenant.
//   - sin pedido: 0 memberships → forbidden; 1 → esa; >1 → se exige selección (nunca elige arbitrario).
//
// Satisface PrincipalVerifier, así que es intercambiable con el ClaimsResolver en el middleware.
type MembershipResolver struct {
	verifier    TokenVerifierPort
	memberships MembershipResolverPort
	cfg         MembershipConfig
}

var _ PrincipalVerifier = (*MembershipResolver)(nil)

func NewMembershipResolver(verifier TokenVerifierPort, memberships MembershipResolverPort, cfg MembershipConfig) *MembershipResolver {
	return &MembershipResolver{verifier: verifier, memberships: memberships, cfg: cfg}
}

func (r *MembershipResolver) Verify(ctx context.Context, bearerToken string) (identitydomain.Principal, error) {
	if r == nil || r.verifier == nil || r.memberships == nil {
		return identitydomain.Principal{}, domainerr.Internal("membership resolver not configured")
	}
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

	actorClaim := r.cfg.ActorClaim
	if actorClaim == "" {
		actorClaim = "sub"
	}
	actor := strings.TrimSpace(toString(claimValue(claims, actorClaim)))
	if actor == "" {
		return identitydomain.Principal{}, domainerr.Unauthorized("missing subject claim")
	}

	rows, err := r.memberships.TenantsForActor(ctx, actor)
	if err != nil {
		return identitydomain.Principal{}, domainerr.Internal("failed resolving memberships")
	}

	chosen, err := selectMembership(rows, RequestedTenantFromContext(ctx))
	if err != nil {
		return identitydomain.Principal{}, err
	}

	return identitydomain.Principal{
		OrgID:      chosen.TenantID,
		TenantID:   chosen.TenantID,
		Actor:      actor,
		Role:       chosen.Role,
		Scopes:     append([]string(nil), chosen.Scopes...),
		AuthMethod: "jwt",
	}, nil
}

// selectMembership aplica la política 0/1/>1. Con tenant pedido explícito, exige membership en ese
// tenant sin importar la cantidad (nunca cae a otro).
func selectMembership(rows []ActorTenant, requestedTenant string) (ActorTenant, error) {
	if requestedTenant != "" {
		for _, row := range rows {
			if row.TenantID == requestedTenant {
				return row, nil
			}
		}
		return ActorTenant{}, domainerr.Forbidden("no membership in requested tenant")
	}
	switch len(rows) {
	case 0:
		return ActorTenant{}, domainerr.Forbidden("no membership for user")
	case 1:
		return rows[0], nil
	default:
		return ActorTenant{}, domainerr.Newf(domainerr.KindValidation, "tenant selection required")
	}
}
