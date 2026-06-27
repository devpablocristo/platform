package clerk

import (
	identity "github.com/devpablocristo/platform/kernels/saas/go/identity"
	identitydomain "github.com/devpablocristo/platform/kernels/saas/go/identity/usecases/domain"
)

const Provider = "clerk"

type ClaimsAdapter struct {
	Mapping identity.ExternalIdentityMapping
}

func DefaultMapping() identity.ExternalIdentityMapping {
	return identity.ExternalIdentityMapping{
		Provider:          Provider,
		ActorClaims:       []string{"sub"},
		EmailClaims:       []string{"email", "primary_email_address"},
		OrgClaims:         []string{"org_id", "o.id"},
		ExternalOrgClaims: []string{"org_id", "o.id"},
		RoleClaims:        []string{"org_role", "role", "o.rol"},
		ScopesClaims:      []string{"org_permissions", "o.per", "scopes"},
		DefaultAuthMethod: Provider,
	}
}

func (a ClaimsAdapter) ExternalIdentity(claims map[string]any) identity.ExternalIdentity {
	return identity.ExternalIdentityFromClaims(claims, identity.MergeExternalIdentityMapping(DefaultMapping(), a.Mapping))
}

func (a ClaimsAdapter) Principal(claims map[string]any) identitydomain.Principal {
	mapping := identity.MergeExternalIdentityMapping(DefaultMapping(), a.Mapping)
	return identity.PrincipalFromExternalIdentity(identity.ExternalIdentityFromClaims(claims, mapping), mapping)
}
