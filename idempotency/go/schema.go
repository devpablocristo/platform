package idempotency

import "embed"

// InitialMigration is the path of the composable PostgreSQL schema migration.
const InitialMigration = "migrations/0001_create_idempotency_records.sql"

// Migrations contains the SQL migrations owned by this module.
//
//go:embed migrations/*.sql
var Migrations embed.FS
