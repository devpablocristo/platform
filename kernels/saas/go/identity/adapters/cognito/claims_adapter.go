package cognito

import (
	identity "github.com/devpablocristo/platform/kernels/saas/go/identity"
	identitydomain "github.com/devpablocristo/platform/kernels/saas/go/identity/usecases/domain"
)

const Provider = "cognito"

type ClaimsAdapter struct {
	Mapping identity.ExternalIdentityMapping
}

func DefaultMapping() identity.ExternalIdentityMapping {
	return identity.ExternalIdentityMapping{
		Provider:          Provider,
		ActorClaims:       []string{"sub"},
		EmailClaims:       []string{"email"},
		OrgClaims:         []string{"org_id"},
		ExternalOrgClaims: []string{"org_id"},
		RoleClaims:        []string{"role"},
		ScopesClaims:      []string{"scope", "scp", "scopes"},
		GroupsClaims:      []string{"cognito:groups"},
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
