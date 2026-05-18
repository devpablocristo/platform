package org

import (
	"context"

	orgdomain "github.com/devpablocristo/core/saas/go/org/usecases/domain"
)

// APIKeyResolver define el puerto de lookup de principal por API key.
type APIKeyResolver interface {
	FindPrincipalByAPIKeyHash(context.Context, string) (orgdomain.Principal, string, error)
}
