package usagemetering

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/devpablocristo/core/observability/go"
	"gorm.io/gorm"
)

type usageRow struct {
	TenantID  string    `gorm:"column:org_id;type:uuid;primaryKey"`
	Period    time.Time `gorm:"type:date;primaryKey"`
	Counter   string    `gorm:"primaryKey"`
	Value     int64     `gorm:"column:value"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (usageRow) TableName() string { return "org_usage_counters" }

type Repository struct {
	db      *gorm.DB
	metrics observability.Sink
	logger  *slog.Logger
}

func NewRepository(db *gorm.DB, sink observability.Sink, logger *slog.Logger) *Repository {
	if logger == nil {
		logger = slog.Default()
	}
	return &Repository{
		db:      db,
		metrics: sink,
		logger:  logger,
	}
}

func (r *Repository) Increment(ctx context.Context, tenantID, counter string) error {
	return r.IncrementEvent(ctx, "", tenantID, counter)
}

func (r *Repository) IncrementEvent(ctx context.Context, eventID, tenantID, counter string) error {
	if r == nil || r.db == nil {
		return errors.New("usagemetering repository not configured")
	}

	now := time.Now().UTC()
	period := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	incremented := false

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if eventID != "" {
			res := tx.Exec(
				`INSERT INTO saas_usage_event_dedup (event_id, org_id, counter, created_at)
				 VALUES (?, ?, ?, now())
				 ON CONFLICT (event_id) DO NOTHING`,
				eventID, tenantID, counter,
			)
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected == 0 {
				return nil
			}
		}

		if err := tx.Exec(
			`INSERT INTO org_usage_counters (org_id, period, counter, value, updated_at)
			 VALUES (?, ?, ?, 1, now())
			 ON CONFLICT (org_id, period, counter)
			 DO UPDATE SET value = org_usage_counters.value + 1, updated_at = now()`,
			tenantID, period, counter,
		).Error; err != nil {
			return err
		}

		incremented = true
		return nil
	})
	if err != nil {
		r.logger.Warn("usage metering increment failed", "tenant_id", tenantID, "counter", counter, "event_id", eventID, "error", err)
		return err
	}
	if incremented && r.metrics != nil {
		r.metrics.IncCounter("usage_metering_events", map[string]string{
			"tenant_id": tenantID,
			"counter":   counter,
		})
	}
	return nil
}

func (r *Repository) GetCounter(ctx context.Context, tenantID, counter string, period time.Time) (int64, error) {
	if r == nil || r.db == nil {
		return 0, errors.New("usagemetering repository not configured")
	}

	period = time.Date(period.UTC().Year(), period.UTC().Month(), 1, 0, 0, 0, 0, time.UTC)
	var row struct {
		Value int64 `gorm:"column:value"`
	}

	err := r.db.WithContext(ctx).
		Table("org_usage_counters").
		Select("value").
		Where("org_id = ? AND period = ? AND counter = ?", tenantID, period.Format("2006-01-02"), counter).
		Take(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, nil
		}
		return 0, err
	}
	return row.Value, nil
}
