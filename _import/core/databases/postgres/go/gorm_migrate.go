package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"hash/fnv"
	"io/fs"
	"log/slog"
	"time"

	"github.com/golang-migrate/migrate/v4"
	pg "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"gorm.io/gorm"

	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// GormMigrateOption configura el runner de migraciones.
type GormMigrateOption func(*gormMigrateConfig)

type gormMigrateConfig struct {
	migrationsTable string
	advisoryLock    bool
	lockTimeout     time.Duration
}

// WithMigrationsTable setea el nombre de la tabla de schema_migrations.
func WithMigrationsTable(name string) GormMigrateOption {
	return func(c *gormMigrateConfig) { c.migrationsTable = name }
}

// WithAdvisoryLock adquiere pg_advisory_lock antes de migrar. Útil en deploys
// con múltiples instancias (Cloud Run, K8s) para evitar migraciones concurrentes.
func WithAdvisoryLock(timeout time.Duration) GormMigrateOption {
	return func(c *gormMigrateConfig) {
		c.advisoryLock = true
		c.lockTimeout = timeout
	}
}

// GormMigrateUp ejecuta migraciones SQL embebidas contra la DB de GORM.
func GormMigrateUp(db *gorm.DB, sqlFiles fs.FS, subdir string, opts ...GormMigrateOption) error {
	if subdir == "" {
		subdir = "."
	}

	cfg := &gormMigrateConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("get sql.DB for migrations: %w", err)
	}

	if cfg.advisoryLock {
		unlock, lockErr := acquireGormAdvisoryLock(context.Background(), sqlDB, cfg.migrationsTable, cfg.lockTimeout)
		if lockErr != nil {
			return fmt.Errorf("acquire migration lock: %w", lockErr)
		}
		defer unlock()
	}

	src, err := iofs.New(sqlFiles, subdir)
	if err != nil {
		return fmt.Errorf("iofs source: %w", err)
	}

	pgCfg := &pg.Config{}
	if cfg.migrationsTable != "" {
		pgCfg.MigrationsTable = cfg.migrationsTable
	}

	driver, err := pg.WithInstance(sqlDB, pgCfg)
	if err != nil {
		return fmt.Errorf("postgres driver: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", src, "postgres", driver)
	if err != nil {
		return fmt.Errorf("new migrate: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("run migrations: %w", err)
	}

	return nil
}

// GormMigrateUpFromDir ejecuta migraciones desde un directorio del filesystem (no embed.FS).
func GormMigrateUpFromDir(db *gorm.DB, dir string, opts ...GormMigrateOption) error {
	cfg := &gormMigrateConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("get sql.DB for migrations: %w", err)
	}

	if cfg.advisoryLock {
		unlock, lockErr := acquireGormAdvisoryLock(context.Background(), sqlDB, cfg.migrationsTable, cfg.lockTimeout)
		if lockErr != nil {
			return fmt.Errorf("acquire migration lock: %w", lockErr)
		}
		defer unlock()
	}

	pgCfg := &pg.Config{}
	if cfg.migrationsTable != "" {
		pgCfg.MigrationsTable = cfg.migrationsTable
	}

	driver, err := pg.WithInstance(sqlDB, pgCfg)
	if err != nil {
		return fmt.Errorf("postgres driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(dir, "postgres", driver)
	if err != nil {
		return fmt.Errorf("new migrate: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("run migrations: %w", err)
	}

	return nil
}

func acquireGormAdvisoryLock(ctx context.Context, sqlDB *sql.DB, name string, timeout time.Duration) (func(), error) {
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}
	lockID := gormHashName(name)
	slog.Info("migration_lock_acquiring", "name", name, "lock_id", lockID)

	deadline := time.Now().Add(timeout)
	for {
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timeout waiting for migration lock (name=%s, lock_id=%d)", name, lockID)
		}
		var acquired bool
		err := sqlDB.QueryRowContext(ctx, "SELECT pg_try_advisory_lock($1)", lockID).Scan(&acquired)
		if err != nil {
			return nil, fmt.Errorf("pg_try_advisory_lock: %w", err)
		}
		if acquired {
			slog.Info("migration_lock_acquired", "name", name, "lock_id", lockID)
			return func() {
				_, _ = sqlDB.ExecContext(context.Background(), "SELECT pg_advisory_unlock($1)", lockID)
				slog.Info("migration_lock_released", "name", name, "lock_id", lockID)
			}, nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}
}

func gormHashName(name string) int64 {
	if name == "" {
		name = "default"
	}
	h := fnv.New64a()
	h.Write([]byte(name))
	return int64(h.Sum64() >> 1)
}
