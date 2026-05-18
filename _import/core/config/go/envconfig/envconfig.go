// Package envconfig provides helpers for reading typed values from environment variables.
package envconfig

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Get devuelve el valor de la variable de entorno o el fallback.
func Get(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// Bool devuelve un booleano desde la variable de entorno o el fallback.
func Bool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return parsed
}

// Int devuelve un entero desde la variable de entorno o el fallback.
func Int(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return parsed
}

// Int64 devuelve un int64 desde la variable de entorno o el fallback.
func Int64(key string, fallback int64) int64 {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	parsed, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

// Float64 devuelve un float64 desde la variable de entorno o el fallback.
func Float64(key string, fallback float64) float64 {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	parsed, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

// Duration devuelve una duración desde la variable de entorno (en segundos) o el fallback.
// El valor de la variable se interpreta como cantidad de segundos (entero).
func Duration(key string, fallback time.Duration) time.Duration {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	sec, err := strconv.Atoi(v)
	if err != nil || sec <= 0 {
		return fallback
	}
	return time.Duration(sec) * time.Second
}

// Required devuelve el valor de la variable o error si está vacía.
func Required(key string) (string, error) {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return "", fmt.Errorf("envconfig: required variable %s is not set", key)
	}
	return v, nil
}

// NormalizeEnv normaliza un nombre de entorno (dev, development, local, test → development).
func NormalizeEnv(value string) string {
	normalized := strings.TrimSpace(strings.ToLower(value))
	if normalized == "" {
		return "development"
	}
	return normalized
}

// IsLocal retorna true para entornos de desarrollo local.
func IsLocal(environment string) bool {
	switch NormalizeEnv(environment) {
	case "development", "dev", "local", "test":
		return true
	default:
		return false
	}
}
