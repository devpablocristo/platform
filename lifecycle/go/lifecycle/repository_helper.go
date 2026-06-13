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

// LifecycleStoreConfig configures the canned SQL implementation for the full
// active/archive/trash/purge lifecycle. All column names are parameters; this
// package does not impose a SQL convention.
//
// PurgeAfterColumn is optional. When empty, Trash records TrashedAtColumn but
// does not persist a purge-after timestamp.
type LifecycleStoreConfig struct {
	Table            string
	IDColumn         string
	TenantColumn     string
	ArchivedAtColumn string
	TrashedAtColumn  string
	PurgeAfterColumn string
}

// LifecycleStore implements RepositoryPort against a *sql.DB.
type LifecycleStore struct {
	db  *sql.DB
	cfg LifecycleStoreConfig
}

// NewLifecycleStore validates cfg and returns a LifecycleStore.
func NewLifecycleStore(db *sql.DB, cfg LifecycleStoreConfig) (*LifecycleStore, error) {
	if db == nil {
		return nil, fmt.Errorf("lifecycle: nil *sql.DB")
	}
	switch {
	case cfg.Table == "":
		return nil, fmt.Errorf("lifecycle: empty LifecycleStoreConfig.Table")
	case cfg.IDColumn == "":
		return nil, fmt.Errorf("lifecycle: empty LifecycleStoreConfig.IDColumn")
	case cfg.TenantColumn == "":
		return nil, fmt.Errorf("lifecycle: empty LifecycleStoreConfig.TenantColumn")
	case cfg.ArchivedAtColumn == "":
		return nil, fmt.Errorf("lifecycle: empty LifecycleStoreConfig.ArchivedAtColumn")
	case cfg.TrashedAtColumn == "":
		return nil, fmt.Errorf("lifecycle: empty LifecycleStoreConfig.TrashedAtColumn")
	}
	return &LifecycleStore{db: db, cfg: cfg}, nil
}

// Archive sets ArchivedAtColumn for active rows only.
func (s *LifecycleStore) Archive(ctx context.Context, tenantID string, resourceID uuid.UUID, at time.Time) error {
	q := fmt.Sprintf(
		`UPDATE %s SET %s = $1 WHERE %s = $2 AND %s = $3 AND %s IS NULL AND %s IS NULL`,
		s.cfg.Table, s.cfg.ArchivedAtColumn,
		s.cfg.IDColumn, s.cfg.TenantColumn, s.cfg.ArchivedAtColumn, s.cfg.TrashedAtColumn,
	)
	return execOne(ctx, s.db, s.cfg.Table, resourceID, q, at.UTC(), resourceID, tenantID)
}

// Unarchive clears ArchivedAtColumn for archived rows.
func (s *LifecycleStore) Unarchive(ctx context.Context, tenantID string, resourceID uuid.UUID) error {
	q := fmt.Sprintf(
		`UPDATE %s SET %s = NULL WHERE %s = $1 AND %s = $2 AND %s IS NOT NULL AND %s IS NULL`,
		s.cfg.Table, s.cfg.ArchivedAtColumn,
		s.cfg.IDColumn, s.cfg.TenantColumn, s.cfg.ArchivedAtColumn, s.cfg.TrashedAtColumn,
	)
	return execOne(ctx, s.db, s.cfg.Table, resourceID, q, resourceID, tenantID)
}

// Trash sets TrashedAtColumn for rows not already trashed. If the row was
// archived, it leaves the archived state and enters the trash state.
func (s *LifecycleStore) Trash(ctx context.Context, tenantID string, resourceID uuid.UUID, at time.Time, purgeAfter *time.Time) error {
	if s.cfg.PurgeAfterColumn == "" {
		q := fmt.Sprintf(
			`UPDATE %s SET %s = NULL, %s = $1 WHERE %s = $2 AND %s = $3 AND %s IS NULL`,
			s.cfg.Table, s.cfg.ArchivedAtColumn, s.cfg.TrashedAtColumn,
			s.cfg.IDColumn, s.cfg.TenantColumn, s.cfg.TrashedAtColumn,
		)
		return execOne(ctx, s.db, s.cfg.Table, resourceID, q, at.UTC(), resourceID, tenantID)
	}
	q := fmt.Sprintf(
		`UPDATE %s SET %s = NULL, %s = $1, %s = $2 WHERE %s = $3 AND %s = $4 AND %s IS NULL`,
		s.cfg.Table, s.cfg.ArchivedAtColumn, s.cfg.TrashedAtColumn, s.cfg.PurgeAfterColumn,
		s.cfg.IDColumn, s.cfg.TenantColumn, s.cfg.TrashedAtColumn,
	)
	return execOne(ctx, s.db, s.cfg.Table, resourceID, q, at.UTC(), nullableTime(purgeAfter), resourceID, tenantID)
}

// Restore clears TrashedAtColumn and PurgeAfterColumn, returning the resource
// to active.
func (s *LifecycleStore) Restore(ctx context.Context, tenantID string, resourceID uuid.UUID) error {
	if s.cfg.PurgeAfterColumn == "" {
		q := fmt.Sprintf(
			`UPDATE %s SET %s = NULL WHERE %s = $1 AND %s = $2 AND %s IS NOT NULL`,
			s.cfg.Table, s.cfg.TrashedAtColumn,
			s.cfg.IDColumn, s.cfg.TenantColumn, s.cfg.TrashedAtColumn,
		)
		return execOne(ctx, s.db, s.cfg.Table, resourceID, q, resourceID, tenantID)
	}
	q := fmt.Sprintf(
		`UPDATE %s SET %s = NULL, %s = NULL WHERE %s = $1 AND %s = $2 AND %s IS NOT NULL`,
		s.cfg.Table, s.cfg.TrashedAtColumn, s.cfg.PurgeAfterColumn,
		s.cfg.IDColumn, s.cfg.TenantColumn, s.cfg.TrashedAtColumn,
	)
	return execOne(ctx, s.db, s.cfg.Table, resourceID, q, resourceID, tenantID)
}

// Purge permanently removes the row.
func (s *LifecycleStore) Purge(ctx context.Context, tenantID string, resourceID uuid.UUID) error {
	q := fmt.Sprintf(
		`DELETE FROM %s WHERE %s = $1 AND %s = $2`,
		s.cfg.Table, s.cfg.IDColumn, s.cfg.TenantColumn,
	)
	return execOne(ctx, s.db, s.cfg.Table, resourceID, q, resourceID, tenantID)
}

// State returns the current lifecycle state for a row.
func (s *LifecycleStore) State(ctx context.Context, tenantID string, resourceID uuid.UUID) (LifecycleState, error) {
	q := fmt.Sprintf(
		`SELECT %s IS NOT NULL, %s IS NOT NULL FROM %s WHERE %s = $1 AND %s = $2`,
		s.cfg.ArchivedAtColumn, s.cfg.TrashedAtColumn,
		s.cfg.Table, s.cfg.IDColumn, s.cfg.TenantColumn,
	)
	var archived, trashed bool
	err := s.db.QueryRowContext(ctx, q, resourceID, tenantID).Scan(&archived, &trashed)
	if errors.Is(err, sql.ErrNoRows) {
		return "", domainerr.NotFoundf(s.cfg.Table, resourceID.String())
	}
	if err != nil {
		return "", err
	}
	switch {
	case trashed:
		return StateTrashed, nil
	case archived:
		return StateArchived, nil
	default:
		return StateActive, nil
	}
}

func execOne(ctx context.Context, db *sql.DB, table string, resourceID uuid.UUID, query string, args ...any) error {
	res, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return domainerr.NotFoundf(table, resourceID.String())
	}
	return nil
}

func nullableTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.UTC()
}

var _ RepositoryPort = (*LifecycleStore)(nil)
