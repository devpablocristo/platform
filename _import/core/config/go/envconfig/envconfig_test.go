package envconfig_test

import (
	"testing"
	"time"

	"github.com/devpablocristo/core/config/go/envconfig"
)

func TestGet(t *testing.T) {
	t.Setenv("TEST_GET_KEY", "value")
	if got := envconfig.Get("TEST_GET_KEY", "default"); got != "value" {
		t.Errorf("expected value, got %s", got)
	}
	if got := envconfig.Get("TEST_GET_MISSING", "fallback"); got != "fallback" {
		t.Errorf("expected fallback, got %s", got)
	}
}

func TestBool(t *testing.T) {
	t.Setenv("TEST_BOOL_TRUE", "true")
	t.Setenv("TEST_BOOL_BAD", "notabool")
	if !envconfig.Bool("TEST_BOOL_TRUE", false) {
		t.Error("expected true")
	}
	if envconfig.Bool("TEST_BOOL_BAD", false) {
		t.Error("expected fallback false for invalid value")
	}
	if !envconfig.Bool("TEST_BOOL_MISSING", true) {
		t.Error("expected fallback true")
	}
}

func TestInt(t *testing.T) {
	t.Setenv("TEST_INT_VAL", "42")
	t.Setenv("TEST_INT_BAD", "abc")
	if got := envconfig.Int("TEST_INT_VAL", 0); got != 42 {
		t.Errorf("expected 42, got %d", got)
	}
	if got := envconfig.Int("TEST_INT_BAD", 99); got != 99 {
		t.Errorf("expected fallback 99, got %d", got)
	}
	if got := envconfig.Int("TEST_INT_MISSING", 10); got != 10 {
		t.Errorf("expected fallback 10, got %d", got)
	}
}

func TestDuration(t *testing.T) {
	t.Setenv("TEST_DUR_SEC", "30")
	t.Setenv("TEST_DUR_BAD", "abc")
	t.Setenv("TEST_DUR_NEG", "-5")
	if got := envconfig.Duration("TEST_DUR_SEC", time.Second); got != 30*time.Second {
		t.Errorf("expected 30s, got %v", got)
	}
	if got := envconfig.Duration("TEST_DUR_BAD", 5*time.Second); got != 5*time.Second {
		t.Errorf("expected fallback 5s, got %v", got)
	}
	if got := envconfig.Duration("TEST_DUR_NEG", 5*time.Second); got != 5*time.Second {
		t.Errorf("expected fallback 5s for negative, got %v", got)
	}
}

func TestRequired(t *testing.T) {
	t.Setenv("TEST_REQ_SET", "ok")
	if _, err := envconfig.Required("TEST_REQ_SET"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if _, err := envconfig.Required("TEST_REQ_MISSING"); err == nil {
		t.Error("expected error for missing required var")
	}
}

func TestIsLocal(t *testing.T) {
	t.Parallel()
	for _, env := range []string{"development", "dev", "local", "test"} {
		if !envconfig.IsLocal(env) {
			t.Errorf("expected %q to be local", env)
		}
	}
	if envconfig.IsLocal("production") {
		t.Error("production should not be local")
	}
	if envconfig.IsLocal("staging") {
		t.Error("staging should not be local")
	}
}
