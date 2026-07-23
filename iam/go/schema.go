package iam

import (
	"embed"

	postgres "github.com/devpablocristo/platform/databases/postgres/go"
)

const (
	// MigrationScope isolates this module's migration versions.
	MigrationScope = "iam/core"
	// MigrationsDir is the embedded migration directory.
	MigrationsDir = "migrations"
)

// Migrations contains the PostgreSQL schema owned by this module.
//
//go:embed migrations/*.sql
var Migrations embed.FS

// MigrationProfile returns this module's composable PostgreSQL migrations.
func MigrationProfile() postgres.MigrationProfile {
	return postgres.MigrationProfile{
		Scope:      MigrationScope,
		Migrations: Migrations,
		Dir:        MigrationsDir,
	}
}
