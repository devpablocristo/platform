package migrations

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"
)

//go:embed *.sql
var files embed.FS

type migrationFile struct {
	version string
	sql     string
}

func MigrateUp(ctx context.Context, db *sql.DB, scope string) error {
	if db == nil {
		return fmt.Errorf("migration database required")
	}
	scope = strings.TrimSpace(scope)
	if scope == "" {
		return fmt.Errorf("migration scope required")
	}

	items, err := loadMigrationFiles(files, ".")
	if err != nil {
		return err
	}

	// Nombre propio: golang-migrate (pymes-core/migrations) ya usa "schema_migrations" con otro esquema.
	if _, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS saas_core_schema_migrations (
			scope text NOT NULL,
			version text NOT NULL,
			applied_at timestamptz NOT NULL,
			PRIMARY KEY (scope, version)
		)
	`); err != nil {
		return fmt.Errorf("ensure saas_core_schema_migrations: %w", err)
	}

	for _, item := range items {
		if err := applyMigration(ctx, db, scope, item); err != nil {
			return fmt.Errorf("apply migration %s/%s: %w", scope, item.version, err)
		}
	}
	return nil
}

func loadMigrationFiles(migrations fs.FS, dir string) ([]migrationFile, error) {
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
		if dir != "" && dir != "." {
			fullPath = path.Join(dir, entry.Name())
		}
		body, err := fs.ReadFile(migrations, fullPath)
		if err != nil {
			return nil, fmt.Errorf("read migration %q: %w", fullPath, err)
		}
		sqlText := strings.TrimSpace(string(body))
		if sqlText == "" {
			return nil, fmt.Errorf("migration %q is empty", fullPath)
		}
		items = append(items, migrationFile{
			version: entry.Name(),
			sql:     sqlText,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].version < items[j].version
	})
	return items, nil
}

func applyMigration(ctx context.Context, db *sql.DB, scope string, item migrationFile) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	result, err := tx.ExecContext(ctx, `
		INSERT INTO saas_core_schema_migrations (scope, version, applied_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (scope, version) DO NOTHING
	`, scope, item.version)
	if err != nil {
		return fmt.Errorf("register migration: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read migration result: %w", err)
	}
	if rowsAffected == 0 {
		return nil
	}
	if _, err := tx.ExecContext(ctx, item.sql); err != nil {
		return fmt.Errorf("execute migration: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit migration: %w", err)
	}
	committed = true
	return nil
}
