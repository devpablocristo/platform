package pagination

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	defaultLimit = 20
	maxLimit     = 100
)

// Config define defaults y techo para paginación basada en cursor.
type Config struct {
	DefaultLimit int
	MaxLimit     int
}

// Params representa parámetros HTTP o internos de paginación.
type Params struct {
	Limit  int    `json:"limit"`
	Cursor string `json:"cursor,omitempty"`
}

// Result representa una página genérica basada en cursor.
type Result[T any] struct {
	Items      []T    `json:"items"`
	HasMore    bool   `json:"has_more"`
	NextCursor string `json:"next_cursor,omitempty"`
}

// TimeIDCursor represents the common stable cursor shape for SQL lists ordered
// by created_at plus an immutable identifier.
type TimeIDCursor struct {
	CreatedAt time.Time `json:"created_at"`
	ID        string    `json:"id"`
}

// DefaultConfig devuelve una configuración razonable para APIs públicas.
func DefaultConfig() Config {
	return Config{
		DefaultLimit: defaultLimit,
		MaxLimit:     maxLimit,
	}
}

// NormalizeConfig aplica defaults seguros.
func NormalizeConfig(config Config) Config {
	if config.DefaultLimit <= 0 {
		config.DefaultLimit = defaultLimit
	}
	if config.MaxLimit <= 0 {
		config.MaxLimit = maxLimit
	}
	if config.DefaultLimit > config.MaxLimit {
		config.DefaultLimit = config.MaxLimit
	}
	return config
}

// NormalizeLimit aplica defaults y techo de paginación.
func NormalizeLimit(limit int, config Config) int {
	config = NormalizeConfig(config)
	switch {
	case limit <= 0:
		return config.DefaultLimit
	case limit > config.MaxLimit:
		return config.MaxLimit
	default:
		return limit
	}
}

// ParseParams normaliza `limit` y `cursor` desde strings HTTP.
func ParseParams(rawLimit, rawCursor string, config Config) (Params, error) {
	config = NormalizeConfig(config)
	limit, err := parsePositiveInt(strings.TrimSpace(rawLimit), config.DefaultLimit)
	if err != nil {
		return Params{}, fmt.Errorf("parse pagination limit: %w", err)
	}
	return Params{
		Limit:  NormalizeLimit(limit, config),
		Cursor: strings.TrimSpace(rawCursor),
	}, nil
}

// BuildResult construye una página copiando items para evitar aliasing accidental.
func BuildResult[T any](items []T, hasMore bool, nextCursor string) Result[T] {
	cloned := append([]T(nil), items...)
	return Result[T]{
		Items:      cloned,
		HasMore:    hasMore,
		NextCursor: strings.TrimSpace(nextCursor),
	}
}

// EncodeTimeIDCursor encodes a created_at + id cursor using base64 raw URL JSON.
func EncodeTimeIDCursor(cursor TimeIDCursor) (string, error) {
	payload := TimeIDCursor{
		CreatedAt: cursor.CreatedAt.UTC(),
		ID:        strings.TrimSpace(cursor.ID),
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

// DecodeTimeIDCursor decodes a cursor encoded by EncodeTimeIDCursor.
func DecodeTimeIDCursor(raw string) (TimeIDCursor, bool, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return TimeIDCursor{}, false, nil
	}
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return TimeIDCursor{}, false, errors.New("invalid cursor")
	}
	var cursor TimeIDCursor
	if err := json.Unmarshal(decoded, &cursor); err != nil {
		return TimeIDCursor{}, false, errors.New("invalid cursor")
	}
	cursor.ID = strings.TrimSpace(cursor.ID)
	cursor.CreatedAt = cursor.CreatedAt.UTC()
	if cursor.ID == "" || cursor.CreatedAt.IsZero() {
		return TimeIDCursor{}, false, errors.New("invalid cursor")
	}
	return cursor, true, nil
}

func parsePositiveInt(raw string, defaultValue int) (int, error) {
	if raw == "" {
		return defaultValue, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return 0, errors.New("invalid positive integer")
	}
	return value, nil
}
