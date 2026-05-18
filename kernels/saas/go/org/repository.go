package org

import (
	"context"

	orgdomain "github.com/devpablocristo/platform/kernels/saas/go/org/usecases/domain"
)

// APIKeyResolver define el puerto de lookup de principal por API key.
type APIKeyResolver interface {
	FindPrincipalByAPIKeyHash(context.Context, string) (orgdomain.Principal, string, error)
}
