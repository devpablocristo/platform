package authn

import (
	"context"
)

// TryInbound autentica un request HTTP típico: intenta JWT (Bearer) primero; si falla o no hay Bearer,
// intenta API key (X-API-Key o Authorization ApiKey). Mismo criterio que saas/go/middleware.AuthMiddleware.
//
// jwtAuth o apiKey pueden ser nil (se omite ese mecanismo). El segundo valor es "jwt" o "api_key".
func TryInbound(ctx context.Context, jwtAuth, apiKey Authenticator, authorization, xAPIKey string) (*Principal, string, error) {
	var jwtErr error
	if jwtAuth != nil {
		if token, ok := BearerToken(authorization); ok {
			p, aerr := jwtAuth.Authenticate(ctx, BearerCredential{Token: token})
			if aerr == nil && p != nil {
				return p, "jwt", nil
			}
			jwtErr = aerr
		}
	}
	if apiKey != nil {
		if key, ok := APIKeyToken(authorization, xAPIKey); ok {
			p, aerr := apiKey.Authenticate(ctx, APIKeyCredential{Key: key})
			if aerr == nil && p != nil {
				return p, "api_key", nil
			}
			return nil, "", aerr
		}
	}
	if jwtErr != nil {
		return nil, "", jwtErr
	}
	return nil, "", ErrNoValidCredential
}
