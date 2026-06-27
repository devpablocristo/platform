package identity

import (
	"strings"

	identitydomain "github.com/devpablocristo/platform/kernels/saas/go/identity/usecases/domain"
)

type ExternalIdentity struct {
	Provider       string   `json:"provider"`
	ExternalUserID string   `json:"external_user_id"`
	Email          string   `json:"email,omitempty"`
	OrgID          string   `json:"org_id,omitempty"`
	ExternalOrgID  string   `json:"external_org_id,omitempty"`
	Role           string   `json:"role,omitempty"`
	Scopes         []string `json:"scopes,omitempty"`
	Groups         []string `json:"groups,omitempty"`
}

type ExternalIdentityMapping struct {
	Provider           string
	ActorClaim         string
	ActorClaims        []string
	EmailClaim         string
	EmailClaims        []string
	OrgClaim           string
	OrgClaims          []string
	ExternalOrgClaim   string
	ExternalOrgClaims  []string
	RoleClaim          string
	RoleClaims         []string
	ScopesClaim        string
	ScopesClaims       []string
	GroupsClaim        string
	GroupsClaims       []string
	DefaultAuthMethod  string
	DefaultPrincipalID string
}

func ExternalIdentityFromClaims(claims map[string]any, mapping ExternalIdentityMapping) ExternalIdentity {
	orgID := firstStringClaim(claims, appendClaimNames(mapping.OrgClaim, mapping.OrgClaims, "org_id")...)
	externalOrgID := firstStringClaim(claims, appendClaimNames(mapping.ExternalOrgClaim, mapping.ExternalOrgClaims)...)
	if externalOrgID == "" {
		externalOrgID = orgID
	}
	groups := firstScopesClaim(claims, appendClaimNames(mapping.GroupsClaim, mapping.GroupsClaims)...)
	role := normalizeRoleValue(firstStringClaim(claims, appendClaimNames(mapping.RoleClaim, mapping.RoleClaims, "role")...))
	if role == "" && len(groups) > 0 {
		role = groups[0]
	}
	return ExternalIdentity{
		Provider:       strings.TrimSpace(mapping.Provider),
		ExternalUserID: firstStringClaim(claims, appendClaimNames(mapping.ActorClaim, mapping.ActorClaims, "sub")...),
		Email:          firstStringClaim(claims, appendClaimNames(mapping.EmailClaim, mapping.EmailClaims, "email")...),
		OrgID:          orgID,
		ExternalOrgID:  externalOrgID,
		Role:           role,
		Scopes:         firstScopesClaim(claims, appendClaimNames(mapping.ScopesClaim, mapping.ScopesClaims, "scopes", "scope", "scp")...),
		Groups:         groups,
	}
}

func PrincipalFromExternalIdentity(identity ExternalIdentity, mapping ExternalIdentityMapping) identitydomain.Principal {
	authMethod := firstNonEmpty(mapping.DefaultAuthMethod, identity.Provider, "external")
	actor := firstNonEmpty(identity.Email, identity.ExternalUserID, mapping.DefaultPrincipalID)
	return identitydomain.Principal{
		OrgID:      identity.OrgID,
		TenantID:   identity.OrgID,
		Actor:      actor,
		Role:       identity.Role,
		Scopes:     append([]string(nil), identity.Scopes...),
		AuthMethod: authMethod,
	}
}

func MergeExternalIdentityMapping(base ExternalIdentityMapping, override ExternalIdentityMapping) ExternalIdentityMapping {
	if override.Provider != "" {
		base.Provider = override.Provider
	}
	if override.ActorClaim != "" || len(override.ActorClaims) > 0 {
		base.ActorClaim = override.ActorClaim
		base.ActorClaims = override.ActorClaims
	}
	if override.EmailClaim != "" || len(override.EmailClaims) > 0 {
		base.EmailClaim = override.EmailClaim
		base.EmailClaims = override.EmailClaims
	}
	if override.OrgClaim != "" || len(override.OrgClaims) > 0 {
		base.OrgClaim = override.OrgClaim
		base.OrgClaims = override.OrgClaims
	}
	if override.ExternalOrgClaim != "" || len(override.ExternalOrgClaims) > 0 {
		base.ExternalOrgClaim = override.ExternalOrgClaim
		base.ExternalOrgClaims = override.ExternalOrgClaims
	}
	if override.RoleClaim != "" || len(override.RoleClaims) > 0 {
		base.RoleClaim = override.RoleClaim
		base.RoleClaims = override.RoleClaims
	}
	if override.ScopesClaim != "" || len(override.ScopesClaims) > 0 {
		base.ScopesClaim = override.ScopesClaim
		base.ScopesClaims = override.ScopesClaims
	}
	if override.GroupsClaim != "" || len(override.GroupsClaims) > 0 {
		base.GroupsClaim = override.GroupsClaim
		base.GroupsClaims = override.GroupsClaims
	}
	if override.DefaultAuthMethod != "" {
		base.DefaultAuthMethod = override.DefaultAuthMethod
	}
	if override.DefaultPrincipalID != "" {
		base.DefaultPrincipalID = override.DefaultPrincipalID
	}
	return base
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func appendClaimNames(primary string, values []string, defaults ...string) []string {
	out := make([]string, 0, 1+len(values)+len(defaults))
	if strings.TrimSpace(primary) != "" {
		out = append(out, primary)
	}
	out = append(out, values...)
	out = append(out, defaults...)
	return out
}
