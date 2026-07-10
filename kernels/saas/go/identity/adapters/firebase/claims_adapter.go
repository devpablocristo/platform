// Package firebase provee soporte para Google Identity Platform / Firebase Authentication como IdP
// del kernel SaaS: un adapter de claims (para deployments que inyectan custom claims de org/role/scopes
// vía setCustomUserClaims) y helpers de verificación (verifier JWKS + ClaimsConfig) en verifier.go.
//
// Los ID tokens de Firebase son OIDC RS256 con issuer https://securetoken.google.com/<project> y
// aud=<project>, verificables por authn/go/jwks contra el endpoint JWK de securetoken. Para el caso
// habitual en que el token NO trae org/role/scopes, usar el resolver-por-membership del paquete
// identity en vez de este adapter/ClaimsConfig.
package firebase

import (
	identity "github.com/devpablocristo/platform/kernels/saas/go/identity"
	identitydomain "github.com/devpablocristo/platform/kernels/saas/go/identity/usecases/domain"
)

const Provider = "firebase"

// DefaultMapping mapea los claims de un ID token de Identity Platform/Firebase al ExternalIdentity.
// El sujeto viene en `sub` (y `user_id`, idéntico). Org/role/scopes se leen de custom claims si el
// deployment los inyecta (setCustomUserClaims); si no, quedan vacíos y se resuelven por membership.
func DefaultMapping() identity.ExternalIdentityMapping {
	return identity.ExternalIdentityMapping{
		Provider:          Provider,
		ActorClaims:       []string{"sub", "user_id"},
		EmailClaims:       []string{"email"},
		OrgClaims:         []string{"org_id", "tenant_id"},
		ExternalOrgClaims: []string{"org_id", "tenant_id"},
		RoleClaims:        []string{"role", "org_role"},
		ScopesClaims:      []string{"scopes", "permissions", "scope"},
		DefaultAuthMethod: Provider,
	}
}

// ClaimsAdapter aplica el mapeo Firebase (con override opcional) sobre claims ya verificados.
type ClaimsAdapter struct {
	Mapping identity.ExternalIdentityMapping
}

func (a ClaimsAdapter) ExternalIdentity(claims map[string]any) identity.ExternalIdentity {
	return identity.ExternalIdentityFromClaims(claims, identity.MergeExternalIdentityMapping(DefaultMapping(), a.Mapping))
}

func (a ClaimsAdapter) Principal(claims map[string]any) identitydomain.Principal {
	mapping := identity.MergeExternalIdentityMapping(DefaultMapping(), a.Mapping)
	return identity.PrincipalFromExternalIdentity(identity.ExternalIdentityFromClaims(claims, mapping), mapping)
}
