// Package tenantctx binds the canonical tenant from context.Context to a
// PostgreSQL transaction-local setting for row-level security policies.
package tenantctx

import (
	"context"
	"errors"
	"fmt"

	"github.com/devpablocristo/platform/security/go/tenant"
	"github.com/jackc/pgx/v5"
)

const (
	// Setting is the business-agnostic PostgreSQL GUC read by tenant RLS
	// policies. Product tables may use any tenant column name.
	Setting = "app.org_id"

	setLocalSQL = "select set_config($1, $2, true)"
)

var ErrNilTransaction = errors.New("postgres tenantctx: transaction is nil")

// SetLocal resolves the canonical tenant fail-closed and stores it only for
// the lifetime of tx. Accepting pgx.Tx intentionally prevents callers from
// applying a local setting to a pool or non-transactional connection.
func SetLocal(ctx context.Context, tx pgx.Tx) error {
	if tx == nil {
		return ErrNilTransaction
	}

	tenantID, err := tenant.Require(ctx)
	if err != nil {
		return fmt.Errorf("postgres tenantctx: resolve tenant: %w", err)
	}
	if _, err := tx.Exec(ctx, setLocalSQL, Setting, tenantID.String()); err != nil {
		return fmt.Errorf("postgres tenantctx: set local tenant: %w", err)
	}
	return nil
}
