package identity

import (
	"context"

	kerneldomain "github.com/devpablocristo/platform/kernels/saas/go/kernel/usecases/domain"
)

// PrincipalVerifier define el puerto de verificación de credenciales.
type PrincipalVerifier interface {
	Verify(context.Context, string) (kerneldomain.Principal, error)
}
