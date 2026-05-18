package postgres

import (
	"os"
	"testing"
	"time"
)

func TestConfigFromEnv(t *testing.T) {
	t.Setenv("CORE_POSTGRES_MIN_CONNS", "2")
	t.Setenv("CORE_POSTGRES_MAX_CONNS", "11")
	t.Setenv("CORE_POSTGRES_MAX_CONN_LIFETIME", "45m")
	t.Setenv("CORE_POSTGRES_MAX_CONN_IDLE_TIME", "3m")
	t.Setenv("CORE_POSTGRES_HEALTH_CHECK_PERIOD", "11s")
	t.Setenv("CORE_POSTGRES_CONNECT_TIMEOUT", "7s")
	t.Setenv("CORE_POSTGRES_STATEMENT_TIMEOUT", "9s")

	config, err := ConfigFromEnv("CORE_POSTGRES", "core-postgres")
	if err != nil {
		t.Fatalf("ConfigFromEnv returned error: %v", err)
	}

	if config.ApplicationName != "core-postgres" ||
		config.MinConns != 2 ||
		config.MaxConns != 11 ||
		config.MaxConnLifetime != 45*time.Minute ||
		config.MaxConnIdleTime != 3*time.Minute ||
		config.HealthCheckPeriod != 11*time.Second ||
		config.ConnectTimeout != 7*time.Second ||
		config.StatementTimeout != 9*time.Second {
		t.Fatalf("unexpected config: %#v", config)
	}
}

func TestConfigFromEnvRejectsInvalidBounds(t *testing.T) {
	t.Setenv("CORE_POSTGRES_MIN_CONNS", "8")
	t.Setenv("CORE_POSTGRES_MAX_CONNS", "4")

	_, err := ConfigFromEnv("CORE_POSTGRES", "core-postgres")
	if err == nil {
		t.Fatal("expected error for invalid bounds")
	}
}

func TestBuildPoolConfigAppliesRuntimeParams(t *testing.T) {
	t.Parallel()

	config := DefaultConfig("core-postgres")
	config.MinConns = 2
	config.MaxConns = 9
	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = 2 * time.Minute
	config.HealthCheckPeriod = 15 * time.Second
	config.ConnectTimeout = 4 * time.Second
	config.StatementTimeout = 6 * time.Second

	poolConfig, err := buildPoolConfig("postgres://postgres:postgres@localhost:5432/core?sslmode=disable", config)
	if err != nil {
		t.Fatalf("buildPoolConfig returned error: %v", err)
	}

	if poolConfig.MinConns != 2 ||
		poolConfig.MaxConns != 9 ||
		poolConfig.MaxConnLifetime != time.Hour ||
		poolConfig.MaxConnIdleTime != 2*time.Minute ||
		poolConfig.HealthCheckPeriod != 15*time.Second ||
		poolConfig.ConnConfig.ConnectTimeout != 4*time.Second ||
		poolConfig.ConnConfig.RuntimeParams["application_name"] != "core-postgres" ||
		poolConfig.ConnConfig.RuntimeParams["statement_timeout"] != "6000" {
		t.Fatalf("unexpected pool config: %#v", poolConfig)
	}
}

func TestConfigFromEnvDefaultsWithoutEnvironment(t *testing.T) {
	unset := []string{
		"CORE_POSTGRES_MIN_CONNS",
		"CORE_POSTGRES_MAX_CONNS",
		"CORE_POSTGRES_MAX_CONN_LIFETIME",
		"CORE_POSTGRES_MAX_CONN_IDLE_TIME",
		"CORE_POSTGRES_HEALTH_CHECK_PERIOD",
		"CORE_POSTGRES_CONNECT_TIMEOUT",
		"CORE_POSTGRES_STATEMENT_TIMEOUT",
	}
	for _, key := range unset {
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("Unsetenv(%s): %v", key, err)
		}
	}

	config, err := ConfigFromEnv("CORE_POSTGRES", "core-postgres")
	if err != nil {
		t.Fatalf("ConfigFromEnv returned error: %v", err)
	}
	if config != DefaultConfig("core-postgres") {
		t.Fatalf("unexpected default config: %#v", config)
	}
}
