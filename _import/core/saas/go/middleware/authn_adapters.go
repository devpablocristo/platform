package middleware

import (
	"context"

	authn "github.com/devpablocristo/core/authn/go"
	"github.com/devpablocristo/core/saas/go/identity"
	kerneldomain "github.com/devpablocristo/core/saas/go/kernel/usecases/domain"
)

// jwtPrincipalVerifier adapta identity.PrincipalVerifier (JWT) a authn.Authenticator.
type jwtPrincipalVerifier struct {
	v identity.PrincipalVerifier
}

func (a jwtPrincipalVerifier) Authenticate(ctx context.Context, cred authn.Credential) (*authn.Principal, error) {
	if a.v == nil {
		return nil, authn.ErrNoValidCredential
	}
	bc, ok := cred.(authn.BearerCredential)
	if !ok {
		return nil, authn.ErrWrongCredentialKind
	}
	kp, err := a.v.Verify(ctx, bc.Token)
	if err != nil {
		return nil, err
	}
	return kernelToAuthnPrincipal(kp), nil
}

// apiKeyPrincipalVerifier adapta identity.PrincipalVerifier (API key) a authn.Authenticator.
type apiKeyPrincipalVerifier struct {
	v identity.PrincipalVerifier
}

func (a apiKeyPrincipalVerifier) Authenticate(ctx context.Context, cred authn.Credential) (*authn.Principal, error) {
	if a.v == nil {
		return nil, authn.ErrNoValidCredential
	}
	kc, ok := cred.(authn.APIKeyCredential)
	if !ok {
		return nil, authn.ErrWrongCredentialKind
	}
	kp, err := a.v.Verify(ctx, kc.Key)
	if err != nil {
		return nil, err
	}
	return kernelToAuthnPrincipal(kp), nil
}

func kernelToAuthnPrincipal(kp kerneldomain.Principal) *authn.Principal {
	return &authn.Principal{
		OrgID:      kp.TenantID,
		Actor:      kp.Actor,
		Role:       kp.Role,
		Scopes:     append([]string(nil), kp.Scopes...),
		AuthMethod: kp.AuthMethod,
	}
}

func authnToKernelPrincipal(p *authn.Principal, fallbackMethod string) kerneldomain.Principal {
	if p == nil {
		return kerneldomain.Principal{}
	}
	method := p.AuthMethod
	if method == "" {
		method = fallbackMethod
	}
	return kerneldomain.Principal{
		TenantID:   p.OrgID,
		Actor:      p.Actor,
		Role:       p.Role,
		Scopes:     append([]string(nil), p.Scopes...),
		AuthMethod: method,
	}
}
