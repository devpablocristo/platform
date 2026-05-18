// Package postgres proporciona inicialización de PostgreSQL/MySQL/SQLite con GORM,
// ping, close, y migración con golang-migrate. Agnóstico al producto.
package postgres

import (
	"context"
	"fmt"
	"time"

	"gorm.io/driver/mysql"
	gormpg "gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DriverType tipo de base de datos.
type DriverType string

const (
	DriverPostgres DriverType = "postgres"
	DriverMySQL    DriverType = "mysql"
	DriverSQLite   DriverType = "sqlite"
)

// GormConfig configuración del pool GORM.
type GormConfig struct {
	Driver          DriverType
	DSN             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
	LogMode         logger.LogLevel
}

// DefaultGormConfig configuración por defecto para Postgres en producción.
func DefaultGormConfig() GormConfig {
	return GormConfig{
		Driver:          DriverPostgres,
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: time.Hour,
		ConnMaxIdleTime: 10 * time.Minute,
		LogMode:         logger.Silent,
	}
}

// OpenGorm abre una conexión GORM con la configuración dada.
// Para Postgres y MySQL: DSN es el connection string.
// Para SQLite: DSN es el path al archivo.
func OpenGorm(dsn string, cfg GormConfig) (*gorm.DB, error) {
	if dsn == "" {
		return nil, fmt.Errorf("database dsn is required")
	}

	dialector, err := buildDialector(cfg.Driver, dsn)
	if err != nil {
		return nil, err
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(cfg.LogMode),
	})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// SQLite no soporta pool settings
	if cfg.Driver == DriverSQLite {
		return db, nil
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql.DB: %w", err)
	}
	if cfg.MaxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns > 0 {
		sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	}
	if cfg.ConnMaxLifetime > 0 {
		sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	}
	if cfg.ConnMaxIdleTime > 0 {
		sqlDB.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)
	}

	return db, nil
}

// OpenGormPostgres abre Postgres con defaults. Shortcut más usado.
func OpenGormPostgres(dsn string) (*gorm.DB, error) {
	return OpenGorm(dsn, DefaultGormConfig())
}

// GormPing verifica que la conexión esté activa.
func GormPing(ctx context.Context, db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("database is nil")
	}
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("get sql.DB: %w", err)
	}
	return sqlDB.PingContext(ctx)
}

// GormClose cierra la conexión.
func GormClose(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func buildDialector(driver DriverType, dsn string) (gorm.Dialector, error) {
	switch driver {
	case DriverPostgres, "":
		return gormpg.Open(dsn), nil
	case DriverMySQL:
		return mysql.Open(dsn), nil
	case DriverSQLite:
		return sqlite.Open(dsn), nil
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", driver)
	}
}
