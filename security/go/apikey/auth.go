// Package apikey provides API key authentication with SHA-256 hashing.
// auth.go contiene la lógica pura de autenticación (sin HTTP).
// middleware.go contiene el adapter HTTP (net/http middleware).
package apikey

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

const (
	// HeaderName es el header preferido para API keys.
	HeaderName = "X-API-Key"
)

type contextKey string

const principalContextKey contextKey = "core.apikey.principal"

// Principal identifica al caller autenticado por API key.
type Principal struct {
	Name string
}

// Authenticator valida API keys contra hashes SHA-256.
type Authenticator struct {
	entries []entry
}

type entry struct {
	name string
	hash [sha256.Size]byte
}

var (
	ErrMissingAPIKeys = errors.New("api key configuration required")
	ErrInvalidConfig  = errors.New("invalid api key configuration")
)

// NewAuthenticator parsea una config como `admin=secret,data=secret2`.
func NewAuthenticator(raw string) (*Authenticator, error) {
	parsed, err := parse(raw)
	if err != nil {
		return nil, err
	}
	if len(parsed) == 0 {
		return nil, ErrMissingAPIKeys
	}
	return &Authenticator{entries: parsed}, nil
}

// Authenticate verifica una API key cruda y retorna el principal si es válida.
func (a *Authenticator) Authenticate(rawKey string) (Principal, bool) {
	if a == nil || strings.TrimSpace(rawKey) == "" {
		return Principal{}, false
	}
	sum := sha256.Sum256([]byte(rawKey))
	for _, item := range a.entries {
		if subtle.ConstantTimeCompare(sum[:], item.hash[:]) == 1 {
			return Principal{Name: item.name}, true
		}
	}
	return Principal{}, false
}

// PrincipalFromContext retorna el principal autenticado si existe.
func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	principal, ok := ctx.Value(principalContextKey).(Principal)
	return principal, ok
}

// Apply escribe la API key en un request saliente.
func Apply(r *http.Request, key string) {
	key = strings.TrimSpace(key)
	if key == "" {
		return
	}
	r.Header.Set(HeaderName, key)
}

// ExtractKey extrae la API key de un request HTTP.
func ExtractKey(r *http.Request) string {
	if value := strings.TrimSpace(r.Header.Get(HeaderName)); value != "" {
		return value
	}
	authz := strings.TrimSpace(r.Header.Get("Authorization"))
	if authz == "" {
		return ""
	}
	const prefix = "Bearer "
	if !strings.HasPrefix(authz, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(authz, prefix))
}

// DebugHash devuelve el SHA-256 de una key en formato hex para diagnóstico.
func DebugHash(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

func parse(raw string) ([]entry, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '\n'
	})
	items := make([]entry, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		piece := strings.TrimSpace(part)
		if piece == "" {
			continue
		}
		name, secret, ok := strings.Cut(piece, "=")
		if !ok {
			return nil, fmt.Errorf("%w: expected name=secret pair", ErrInvalidConfig)
		}
		name = strings.TrimSpace(name)
		secret = strings.TrimSpace(secret)
		if name == "" || secret == "" {
			return nil, fmt.Errorf("%w: empty name or secret", ErrInvalidConfig)
		}
		if _, exists := seen[name]; exists {
			return nil, fmt.Errorf("%w: duplicate key name %q", ErrInvalidConfig, name)
		}
		seen[name] = struct{}{}
		sum := sha256.Sum256([]byte(secret))
		items = append(items, entry{name: name, hash: sum})
	}
	if len(items) == 0 {
		return nil, ErrMissingAPIKeys
	}
	return items, nil
}
