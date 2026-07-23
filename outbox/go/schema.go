package outbox

import (
	"embed"
	"strings"

	postgres "github.com/devpablocristo/platform/databases/postgres/go"
	"github.com/jackc/pgx/v5"
)

const (
	// DefaultTableName is the table installed by MigrationProfile.
	DefaultTableName = "platform_outbox_messages"
	// MigrationScope isolates this module's migration versions.
	MigrationScope = "outbox/core"
	// MigrationsDir is the root of the embedded migration filesystem.
	MigrationsDir = "."
)

//go:embed schema.sql
var defaultSchemaSQL string

// Migrations contains the default PostgreSQL outbox schema.
//
//go:embed schema.sql
var Migrations embed.FS

// MigrationProfile returns this module's composable PostgreSQL migrations.
//
// Consumers that need a custom table name can continue to use SchemaSQL.
func MigrationProfile() postgres.MigrationProfile {
	return postgres.MigrationProfile{
		Scope:      MigrationScope,
		Migrations: Migrations,
		Dir:        MigrationsDir,
	}
}

// SchemaSQL returns idempotent DDL for table. An empty identifier selects
// DefaultTableName. A schema-qualified identifier is supported.
func SchemaSQL(table pgx.Identifier) (string, error) {
	normalized, err := normalizeTable(table)
	if err != nil {
		return "", err
	}

	tableSQL := normalized.Sanitize()
	baseName := normalized[len(normalized)-1]
	indexSQL := pgx.Identifier{baseName + "_ready_idx"}.Sanitize()
	attemptsConstraintSQL := pgx.Identifier{baseName + "_attempts_valid"}.Sanitize()
	headersConstraintSQL := pgx.Identifier{baseName + "_headers_object"}.Sanitize()
	leaseConstraintSQL := pgx.Identifier{baseName + "_lease_consistent"}.Sanitize()
	terminalConstraintSQL := pgx.Identifier{baseName + "_terminal_state"}.Sanitize()
	rendered := strings.ReplaceAll(defaultSchemaSQL, `"platform_outbox_messages_ready_idx"`, indexSQL)
	rendered = strings.ReplaceAll(rendered, "platform_outbox_attempts_valid", attemptsConstraintSQL)
	rendered = strings.ReplaceAll(rendered, "platform_outbox_headers_object", headersConstraintSQL)
	rendered = strings.ReplaceAll(rendered, "platform_outbox_lease_consistent", leaseConstraintSQL)
	rendered = strings.ReplaceAll(rendered, "platform_outbox_terminal_state", terminalConstraintSQL)
	rendered = strings.ReplaceAll(rendered, `"platform_outbox_messages"`, tableSQL)
	return rendered, nil
}

func normalizeTable(table pgx.Identifier) (pgx.Identifier, error) {
	if len(table) == 0 {
		return pgx.Identifier{DefaultTableName}, nil
	}
	if len(table) > 2 {
		return nil, ErrInvalidTable
	}
	normalized := make(pgx.Identifier, len(table))
	for index, part := range table {
		part = strings.TrimSpace(part)
		if part == "" || strings.ContainsRune(part, '\x00') {
			return nil, ErrInvalidTable
		}
		normalized[index] = part
	}
	return normalized, nil
}
