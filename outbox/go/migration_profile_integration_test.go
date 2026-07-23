package outbox

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	postgres "github.com/devpablocristo/platform/databases/postgres/go"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestMigrationProfileAppliesIdempotently(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("PLATFORM_POSTGRES_TEST_DSN"))
	if dsn == "" {
		if strings.EqualFold(os.Getenv("CI"), "true") {
			t.Fatal("PLATFORM_POSTGRES_TEST_DSN is required in CI")
		}
		t.Skip("PLATFORM_POSTGRES_TEST_DSN is not set")
	}

	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()

	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect PostgreSQL: %v", err)
	}
	defer admin.Close()

	schemaName := fmt.Sprintf("outbox_profile_%d", time.Now().UnixNano())
	schemaSQL := pgx.Identifier{schemaName}.Sanitize()
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+schemaSQL); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		_, _ = admin.Exec(cleanupCtx, "DROP SCHEMA IF EXISTS "+schemaSQL+" CASCADE")
	}()

	scopedURL, err := url.Parse(dsn)
	if err != nil {
		t.Fatalf("parse PostgreSQL URL: %v", err)
	}
	query := scopedURL.Query()
	query.Set("search_path", schemaName)
	scopedURL.RawQuery = query.Encode()

	database, err := postgres.Open(ctx, scopedURL.String())
	if err != nil {
		t.Fatalf("connect scoped PostgreSQL database: %v", err)
	}
	defer database.Close()

	for attempt := 1; attempt <= 2; attempt++ {
		if err := postgres.MigrateProfiles(ctx, database, MigrationProfile()); err != nil {
			t.Fatalf("migrate profile attempt %d: %v", attempt, err)
		}
	}

	pool := database.Pool()
	var tableCount int
	if err := pool.QueryRow(ctx, `
		SELECT count(*)
		FROM information_schema.tables
		WHERE table_schema = $1 AND table_name = $2
	`, schemaName, DefaultTableName).Scan(&tableCount); err != nil {
		t.Fatalf("inspect outbox table: %v", err)
	}
	if tableCount != 1 {
		t.Fatalf("outbox table count = %d, want 1", tableCount)
	}

	var migrationCount int
	if err := pool.QueryRow(ctx, `
		SELECT count(*)
		FROM schema_migrations
		WHERE scope = $1 AND version = 'schema.sql'
	`, MigrationScope).Scan(&migrationCount); err != nil {
		t.Fatalf("inspect migration record: %v", err)
	}
	if migrationCount != 1 {
		t.Fatalf("migration record count = %d, want 1", migrationCount)
	}
}
