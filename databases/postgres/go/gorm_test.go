package postgres

import (
	"testing"
	"time"
)

func TestDefaultGormConfig(t *testing.T) {
	t.Parallel()
	cfg := DefaultGormConfig()
	if cfg.MaxOpenConns != 10 {
		t.Errorf("expected MaxOpenConns=10, got %d", cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns != 5 {
		t.Errorf("expected MaxIdleConns=5, got %d", cfg.MaxIdleConns)
	}
	if cfg.ConnMaxLifetime != time.Hour {
		t.Errorf("expected ConnMaxLifetime=1h, got %v", cfg.ConnMaxLifetime)
	}
}

func TestOpenGormEmptyURL(t *testing.T) {
	t.Parallel()
	_, err := OpenGorm("", DefaultGormConfig())
	if err == nil {
		t.Fatal("expected error on empty URL")
	}
}

func TestGormPingNilDB(t *testing.T) {
	t.Parallel()
	err := GormPing(nil, nil)
	if err == nil {
		t.Fatal("expected error on nil DB")
	}
}

func TestGormCloseNilDB(t *testing.T) {
	t.Parallel()
	err := GormClose(nil)
	if err != nil {
		t.Fatalf("expected no error on nil DB, got %v", err)
	}
}
