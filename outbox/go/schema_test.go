package outbox

import (
	"errors"
	"io/fs"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

func TestMigrationProfileUsesDefaultSchema(t *testing.T) {
	t.Parallel()

	profile := MigrationProfile()
	if profile.Scope != MigrationScope || profile.Dir != MigrationsDir {
		t.Fatalf("unexpected migration profile: %#v", profile)
	}
	body, err := fs.ReadFile(profile.Migrations, "schema.sql")
	if err != nil {
		t.Fatalf("read embedded migration: %v", err)
	}

	sql := string(body)
	for _, required := range []string{
		`CREATE TABLE IF NOT EXISTS "platform_outbox_messages"`,
		"idempotency_key TEXT NOT NULL UNIQUE",
		"lease_token TEXT",
		"lease_expires_at TIMESTAMPTZ",
	} {
		if !strings.Contains(sql, required) {
			t.Errorf("migration does not contain %q", required)
		}
	}
	for _, forbidden := range []string{"pymes", "clerk", "iam.", "organization"} {
		if strings.Contains(strings.ToLower(sql), forbidden) {
			t.Errorf("migration contains product/provider concept %q", forbidden)
		}
	}
}

func TestSchemaSQLDefaultAndQualifiedTable(t *testing.T) {
	t.Parallel()

	defaultSQL, err := SchemaSQL(nil)
	if err != nil {
		t.Fatalf("SchemaSQL(default): %v", err)
	}
	if !strings.Contains(defaultSQL, `CREATE TABLE IF NOT EXISTS "platform_outbox_messages"`) {
		t.Fatalf("default schema does not contain default table: %s", defaultSQL)
	}
	if strings.Contains(defaultSQL, "platform_outbox_messages_messages") {
		t.Fatalf("default schema duplicated the table name: %s", defaultSQL)
	}
	if !strings.Contains(defaultSQL, "attempts <= max_attempts") {
		t.Fatalf("default schema does not enforce the attempt ceiling: %s", defaultSQL)
	}

	customSQL, err := SchemaSQL(pgx.Identifier{"events", "messages"})
	if err != nil {
		t.Fatalf("SchemaSQL(custom): %v", err)
	}
	for _, expected := range []string{
		`CREATE TABLE IF NOT EXISTS "events"."messages"`,
		`CONSTRAINT "messages_attempts_valid"`,
		`CREATE INDEX IF NOT EXISTS "messages_ready_idx"`,
		`ON "events"."messages"`,
	} {
		if !strings.Contains(customSQL, expected) {
			t.Errorf("custom schema missing %q: %s", expected, customSQL)
		}
	}
}

func TestSchemaSQLRejectsInvalidTable(t *testing.T) {
	t.Parallel()

	for _, table := range []pgx.Identifier{{""}, {"one", "two", "three"}, {"bad\x00name"}} {
		if _, err := SchemaSQL(table); !errors.Is(err, ErrInvalidTable) {
			t.Errorf("SchemaSQL(%q) error = %v, want ErrInvalidTable", table, err)
		}
	}
}
