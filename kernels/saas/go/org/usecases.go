package org

import (
	"context"
	"strings"

	orgdomain "github.com/devpablocristo/platform/kernels/saas/go/org/usecases/domain"
)

type UseCases struct {
	resolver APIKeyResolver
}

func NewUseCases(resolver APIKeyResolver) *UseCases {
	return &UseCases{resolver: resolver}
}

func (u *UseCases) ResolvePrincipal(ctx context.Context, apiKeyHash string) (orgdomain.Principal, error) {
	principal, _, err := u.resolver.FindPrincipalByAPIKeyHash(ctx, strings.TrimSpace(apiKeyHash))
	return principal, err
}
