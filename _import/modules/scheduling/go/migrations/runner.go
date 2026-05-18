package migrations

import (
	"embed"
	"io/fs"

	gormdb "github.com/devpablocristo/core/databases/postgres/go"
	"gorm.io/gorm"
)

const DefaultMigrationsTable = "modules_scheduling_schema_migrations"

//go:embed *.sql
var sqlFiles embed.FS

func Run(db *gorm.DB) error {
	return RunWithTable(db, DefaultMigrationsTable)
}

func RunWithTable(db *gorm.DB, migrationsTable string) error {
	return gormdb.GormMigrateUp(db, sqlFiles, ".", gormdb.WithMigrationsTable(migrationsTable))
}

// SQLFiles expone el embed.FS con las migraciones como fs.FS para que los
// consumidores puedan componer runners custom (por ejemplo, interleavear con
// migraciones de otro módulo que tienen FKs cruzadas).
func SQLFiles() fs.FS {
	return sqlFiles
}
