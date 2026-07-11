package postgres

import (
	"context"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
)

type migrationTx interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

type migrationRow interface {
	Scan(dest ...any) error
}

type migrationDB interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) migrationRow
	Begin(ctx context.Context) (migrationTx, error)
}

type migrationFile struct {
	version       string
	sql           string
	transactional bool
}

const nonTransactionalMigrationDirective = "-- platform:migrate:non-transactional"

// Exec ejecuta SQL arbitrario sobre el pool.
func (db *DB) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	if db == nil || db.pool == nil {
		return pgconn.CommandTag{}, fmt.Errorf("postgres pool is nil")
	}
	return db.pool.Exec(ctx, sql, args...)
}

// QueryRow ejecuta una consulta que retorna una fila sobre el pool.
func (db *DB) QueryRow(ctx context.Context, sql string, args ...any) migrationRow {
	if db == nil || db.pool == nil {
		return migrationErrorRow{err: fmt.Errorf("postgres pool is nil")}
	}
	return db.pool.QueryRow(ctx, sql, args...)
}

type migrationErrorRow struct {
	err error
}

func (row migrationErrorRow) Scan(...any) error {
	return row.err
}

// Begin inicia una transacción apta para migraciones.
func (db *DB) Begin(ctx context.Context) (migrationTx, error) {
	if db == nil || db.pool == nil {
		return nil, fmt.Errorf("postgres pool is nil")
	}
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return tx, nil
}

// MigrateUp aplica migraciones `.sql` en orden lexicográfico dentro de un scope.
func MigrateUp(ctx context.Context, db migrationDB, scope string, migrations fs.FS, dir string) error {
	scope = strings.TrimSpace(scope)
	if scope == "" {
		return fmt.Errorf("migration scope required")
	}

	items, err := loadMigrationFiles(migrations, dir)
	if err != nil {
		return err
	}

	if _, err := db.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			scope text NOT NULL,
			version text NOT NULL,
			applied_at timestamptz NOT NULL,
			PRIMARY KEY (scope, version)
		)
	`); err != nil {
		return fmt.Errorf("ensure schema_migrations: %w", err)
	}

	for _, item := range items {
		if err := applyMigration(ctx, db, scope, item); err != nil {
			return fmt.Errorf("apply migration %s/%s: %w", scope, item.version, err)
		}
	}
	return nil
}

func loadMigrationFiles(migrations fs.FS, dir string) ([]migrationFile, error) {
	dir = strings.Trim(strings.TrimSpace(dir), "/")
	entries, err := fs.ReadDir(migrations, dir)
	if err != nil {
		return nil, fmt.Errorf("read migrations dir %q: %w", dir, err)
	}

	items := make([]migrationFile, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		fullPath := entry.Name()
		if dir != "" {
			fullPath = path.Join(dir, entry.Name())
		}
		body, err := fs.ReadFile(migrations, fullPath)
		if err != nil {
			return nil, fmt.Errorf("read migration %q: %w", fullPath, err)
		}
		sql, transactional := parseMigrationSQL(string(body))
		if sql == "" {
			return nil, fmt.Errorf("migration %q is empty", fullPath)
		}
		items = append(items, migrationFile{
			version:       entry.Name(),
			sql:           sql,
			transactional: transactional,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].version < items[j].version
	})
	return items, nil
}

func applyMigration(ctx context.Context, db migrationDB, scope string, item migrationFile) error {
	if !item.transactional {
		return applyNonTransactionalMigration(ctx, db, scope, item)
	}

	tx, err := db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	tag, err := tx.Exec(ctx, `
		INSERT INTO schema_migrations (scope, version, applied_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (scope, version) DO NOTHING
	`, scope, item.version)
	if err != nil {
		return fmt.Errorf("register migration: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil
	}

	if _, err := tx.Exec(ctx, item.sql); err != nil {
		return fmt.Errorf("execute migration: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit migration: %w", err)
	}
	committed = true
	return nil
}

func applyNonTransactionalMigration(ctx context.Context, db migrationDB, scope string, item migrationFile) error {
	applied, err := migrationApplied(ctx, db, scope, item.version)
	if err != nil {
		return fmt.Errorf("check migration: %w", err)
	}
	if applied {
		return nil
	}

	if _, err := db.Exec(ctx, item.sql); err != nil {
		return fmt.Errorf("execute migration: %w", err)
	}
	if _, err := db.Exec(ctx, `
		INSERT INTO schema_migrations (scope, version, applied_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (scope, version) DO NOTHING
	`, scope, item.version); err != nil {
		return fmt.Errorf("register migration: %w", err)
	}
	return nil
}

func migrationApplied(ctx context.Context, db migrationDB, scope, version string) (bool, error) {
	var applied bool
	err := db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM schema_migrations
			WHERE scope = $1 AND version = $2
		)
	`, scope, version).Scan(&applied)
	return applied, err
}

func parseMigrationSQL(raw string) (string, bool) {
	sql := strings.TrimSpace(raw)
	if sql == "" {
		return "", true
	}

	lines := strings.Split(sql, "\n")
	firstLine := strings.TrimSpace(lines[0])
	if firstLine != nonTransactionalMigrationDirective {
		return sql, true
	}
	return strings.TrimSpace(strings.Join(lines[1:], "\n")), false
}
