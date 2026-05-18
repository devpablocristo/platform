package authn

import (
	"context"
	"errors"
)

// APIKeyResolveFunc resuelve una API key cruda a Principal (lookup en DB, HMAC, etc.).
type APIKeyResolveFunc func(ctx context.Context, rawKey string) (*Principal, error)

// APIKeyFuncAuthenticator implementa Authenticator solo para APIKeyCredential.
type APIKeyFuncAuthenticator struct {
	Resolve APIKeyResolveFunc
}

// Authenticate implementa Authenticator.
func (a *APIKeyFuncAuthenticator) Authenticate(ctx context.Context, cred Credential) (*Principal, error) {
	if a == nil || a.Resolve == nil {
		return nil, errors.New("authn: nil api key authenticator")
	}
	if cred == nil || cred.Kind() != KindAPIKey {
		return nil, ErrWrongCredentialKind
	}
	kc, ok := cred.(APIKeyCredential)
	if !ok {
		return nil, ErrWrongCredentialKind
	}
	return a.Resolve(ctx, kc.Key)
}
