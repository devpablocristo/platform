package identity

import (
	"context"
	"errors"

	authn "github.com/devpablocristo/core/authn/go"
	identitydomain "github.com/devpablocristo/core/saas/go/identity/usecases/domain"
)

var ErrVerifierRequired = errors.New("principal verifier is required")

type Config struct {
	Issuer      string
	Audience    string
	OrgClaim    string
	RoleClaim   string
	ScopesClaim string
	ActorClaim  string
}

type legacyOrgResolverPort interface {
	FindOrgIDByExternalID(context.Context, string) (string, bool, error)
}

type legacyOrgResolverAdapter struct {
	inner legacyOrgResolverPort
}

func (a legacyOrgResolverAdapter) FindTenantIDByExternalID(ctx context.Context, externalID string) (string, bool, error) {
	return a.inner.FindOrgIDByExternalID(ctx, externalID)
}

// UseCases agrupa parsing y verificación de identidad reusable.
type UseCases struct {
	verifier PrincipalVerifier
	resolver *ClaimsResolver
}

func NewUseCases(verifier PrincipalVerifier) *UseCases {
	return &UseCases{verifier: verifier}
}

func NewUsecases(verifier TokenVerifierPort, cfg Config) *UseCases {
	return NewUsecasesWithOrgResolver(verifier, nil, cfg)
}

func NewUsecasesWithOrgResolver(verifier TokenVerifierPort, orgs legacyOrgResolverPort, cfg Config) *UseCases {
	var resolverPort OrgResolverPort
	if orgs != nil {
		resolverPort = legacyOrgResolverAdapter{inner: orgs}
	}
	resolver := NewClaimsResolverWithOrgResolver(verifier, resolverPort, ClaimsConfig{
		Issuer:      cfg.Issuer,
		Audience:    cfg.Audience,
		TenantClaim: cfg.OrgClaim,
		RoleClaim:   cfg.RoleClaim,
		ScopesClaim: cfg.ScopesClaim,
		ActorClaim:  cfg.ActorClaim,
	})
	return &UseCases{
		verifier: resolver,
		resolver: resolver,
	}
}

func (u *UseCases) Verify(token string) (identitydomain.Principal, error) {
	if u == nil || u.verifier == nil {
		return identitydomain.Principal{}, ErrVerifierRequired
	}
	return u.verifier.Verify(context.Background(), token)
}

func (u *UseCases) ResolvePrincipal(ctx context.Context, token string) (identitydomain.Principal, error) {
	if u == nil {
		return identitydomain.Principal{}, ErrVerifierRequired
	}
	if u.resolver != nil {
		return u.resolver.ResolvePrincipal(ctx, token)
	}
	if u.verifier == nil {
		return identitydomain.Principal{}, ErrVerifierRequired
	}
	return u.verifier.Verify(ctx, token)
}

func (u *UseCases) BearerToken(raw string) (string, bool) {
	return authn.BearerToken(raw)
}

func (u *UseCases) APIKeyToken(authHeader, xAPIKey string) (string, bool) {
	return authn.APIKeyToken(authHeader, xAPIKey)
}
