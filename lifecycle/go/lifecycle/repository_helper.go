package lifecycle

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/devpablocristo/platform/errors/go/domainerr"
	"github.com/google/uuid"
)

// SoftDeleterConfig configures the canned SQL implementation. All column
// names are parameters — this package does not impose a SQL convention.
//
// Example for a pymes table where the convention is `archived_at`:
//
//	cfg := lifecycle.SoftDeleterConfig{
//	    Table:            "customers",
//	    IDColumn:         "id",
//	    TenantColumn:     "org_id",
//	    ArchivedAtColumn: "archived_at",
//	}
//
// For a table using `deleted_at` instead, ArchivedAtColumn = "deleted_at".
type SoftDeleterConfig struct {
	Table            string
	IDColumn         string
	TenantColumn     string
	ArchivedAtColumn string
}

// SoftDeleter implements RepositoryPort against a *sql.DB using the column
// names from SoftDeleterConfig. Built for the simple case where the resource
// lives in a single table; consumers with more elaborate schemas (cascade
// archive across tables, additional invariants) should implement
// RepositoryPort directly.
type SoftDeleter struct {
	db  *sql.DB
	cfg SoftDeleterConfig
}

// NewSoftDeleter validates cfg and returns a SoftDeleter. Returns an error
// if any required column name is empty.
func NewSoftDeleter(db *sql.DB, cfg SoftDeleterConfig) (*SoftDeleter, error) {
	if db == nil {
		return nil, fmt.Errorf("lifecycle: nil *sql.DB")
	}
	switch {
	case cfg.Table == "":
		return nil, fmt.Errorf("lifecycle: empty SoftDeleterConfig.Table")
	case cfg.IDColumn == "":
		return nil, fmt.Errorf("lifecycle: empty SoftDeleterConfig.IDColumn")
	case cfg.TenantColumn == "":
		return nil, fmt.Errorf("lifecycle: empty SoftDeleterConfig.TenantColumn")
	case cfg.ArchivedAtColumn == "":
		return nil, fmt.Errorf("lifecycle: empty SoftDeleterConfig.ArchivedAtColumn")
	}
	return &SoftDeleter{db: db, cfg: cfg}, nil
}

// SoftDelete sets ArchivedAtColumn to `at` for the matching row.
// Returns domainerr.NotFound if no row matched (tenantID + resourceID combo
// not found or already archived).
func (s *SoftDeleter) SoftDelete(ctx context.Context, tenantID, resourceID uuid.UUID, at time.Time) error {
	q := fmt.Sprintf(
		`UPDATE %s SET %s = $1 WHERE %s = $2 AND %s = $3 AND %s IS NULL`,
		s.cfg.Table, s.cfg.ArchivedAtColumn,
		s.cfg.IDColumn, s.cfg.TenantColumn, s.cfg.ArchivedAtColumn,
	)
	res, err := s.db.ExecContext(ctx, q, at.UTC(), resourceID, tenantID)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return domainerr.NotFoundf(s.cfg.Table, resourceID.String())
	}
	return nil
}

// Restore clears ArchivedAtColumn on the matching row.
// Returns domainerr.NotFound if no archived row matched.
func (s *SoftDeleter) Restore(ctx context.Context, tenantID, resourceID uuid.UUID) error {
	q := fmt.Sprintf(
		`UPDATE %s SET %s = NULL WHERE %s = $1 AND %s = $2 AND %s IS NOT NULL`,
		s.cfg.Table, s.cfg.ArchivedAtColumn,
		s.cfg.IDColumn, s.cfg.TenantColumn, s.cfg.ArchivedAtColumn,
	)
	res, err := s.db.ExecContext(ctx, q, resourceID, tenantID)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return domainerr.NotFoundf(s.cfg.Table, resourceID.String())
	}
	return nil
}

// HardDelete removes the row regardless of its archived state.
func (s *SoftDeleter) HardDelete(ctx context.Context, tenantID, resourceID uuid.UUID) error {
	q := fmt.Sprintf(
		`DELETE FROM %s WHERE %s = $1 AND %s = $2`,
		s.cfg.Table, s.cfg.IDColumn, s.cfg.TenantColumn,
	)
	res, err := s.db.ExecContext(ctx, q, resourceID, tenantID)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return domainerr.NotFoundf(s.cfg.Table, resourceID.String())
	}
	return nil
}

// IsArchived returns true when the row exists and its ArchivedAtColumn is not
// NULL. Returns (false, domainerr.NotFound) when the row does not exist.
func (s *SoftDeleter) IsArchived(ctx context.Context, tenantID, resourceID uuid.UUID) (bool, error) {
	q := fmt.Sprintf(
		`SELECT %s IS NOT NULL FROM %s WHERE %s = $1 AND %s = $2`,
		s.cfg.ArchivedAtColumn, s.cfg.Table, s.cfg.IDColumn, s.cfg.TenantColumn,
	)
	var archived bool
	err := s.db.QueryRowContext(ctx, q, resourceID, tenantID).Scan(&archived)
	if errors.Is(err, sql.ErrNoRows) {
		return false, domainerr.NotFoundf(s.cfg.Table, resourceID.String())
	}
	if err != nil {
		return false, err
	}
	return archived, nil
}

// Ensure SoftDeleter implements RepositoryPort at compile time.
var _ RepositoryPort = (*SoftDeleter)(nil)
