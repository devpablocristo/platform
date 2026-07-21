package outbox

import (
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

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
