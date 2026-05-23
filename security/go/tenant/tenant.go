// Package tenant provee la primitiva canónica de tenant id que platform
// consumers usan en boundaries (HTTP, JWT, repos GORM). Centraliza:
//
//   - tipo opaco ID con conversiones a/desde string y uuid.UUID;
//   - propagación por context.Context — escribiendo a ambas claves
//     (`org_id` y `tenant_id`) de `security/go/contextkeys` para que cualquier
//     consumer reading raw las encuentre;
//   - fail-closed `Require` que retorna `domainerr.TenantMissing()` cuando
//     el contexto no trae tenant — alineado al patrón ponti `MaybeTenantScope`.
//
// Diseño:
//   - ID es `type ID string` (no `uuid.UUID`) porque axis services usan strings
//     no-UUID. Ponti y pymes convierten con `.UUID()` en el border.
//   - strict mode lee `TENANT_STRICT_MODE` env var una vez vía sync.Once.
//     `SetStrictMode(bool)` queda para tests; no es concurrency-safe en runtime.
//   - `WithID` escribe a `OrgID` y `TenantID` keys simultáneamente para
//     compatibilidad con consumers que leen una u otra (ponti = OrgID,
//     pymes = ambas, kernels/saas = TenantID).
package tenant

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync"

	"github.com/google/uuid"

	"github.com/devpablocristo/platform/errors/go/domainerr"
	ctxkeys "github.com/devpablocristo/platform/security/go/contextkeys"
)

// ID es el identificador canónico de tenant en boundaries de platform.
// String form se elige sobre UUID porque axis services manejan tenants
// no-UUID (e.g. "local-dev-org"). Convertir a UUID es cheap y explícito.
type ID string

// String retorna la representación textual.
func (id ID) String() string { return string(id) }

// IsZero indica si la ID es vacía después de trim.
func (id ID) IsZero() bool { return strings.TrimSpace(string(id)) == "" }

// UUID parsea la ID como uuid.UUID. Falla si la string no es un UUID válido.
// Usado por ponti/pymes que tienen `tenant_id uuid` en sus tablas.
func (id ID) UUID() (uuid.UUID, error) {
	if id.IsZero() {
		return uuid.Nil, errors.New("tenant id is empty")
	}
	return uuid.Parse(string(id))
}

// FromUUID construye una ID desde un uuid.UUID. uuid.Nil retorna ID vacía.
func FromUUID(u uuid.UUID) ID {
	if u == uuid.Nil {
		return ""
	}
	return ID(u.String())
}

// FromString construye una ID, trimeando whitespace.
func FromString(s string) ID { return ID(strings.TrimSpace(s)) }

// WithID retorna un contexto que lleva el tenant id en las claves canónicas
// `org_id` y `tenant_id` de `platform/security/go/contextkeys`. Escribir a
// ambas mantiene compat con consumers que leen una u otra directamente.
//
// Si la id es vacía, retorna el contexto sin cambios (no contamina el ctx
// con un valor inválido).
func WithID(ctx context.Context, id ID) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if id.IsZero() {
		return ctx
	}
	s := id.String()
	ctx = context.WithValue(ctx, ctxkeys.OrgID, s)
	ctx = context.WithValue(ctx, ctxkeys.TenantID, s)
	return ctx
}

// FromContext extrae la ID. Soft: retorna (ID, false) si no hay tenant.
// Lee TenantID primero (más específico, kernels/saas convention), luego
// OrgID (axis convention). También tolera valores tipo `uuid.UUID` o
// `fmt.Stringer` para flex con productos que escriben tipos custom.
func FromContext(ctx context.Context) (ID, bool) {
	if ctx == nil {
		return "", false
	}
	for _, key := range []ctxkeys.Key{ctxkeys.TenantID, ctxkeys.OrgID} {
		v := ctx.Value(key)
		if v == nil {
			continue
		}
		if id := coerce(v); !id.IsZero() {
			return id, true
		}
	}
	return "", false
}

// Require retorna la ID o `domainerr.TenantMissing()` si el contexto no
// tiene tenant. Es el helper fail-closed que callers de seguridad usan
// como gate antes de cualquier query tenant-scoped.
func Require(ctx context.Context) (ID, error) {
	id, ok := FromContext(ctx)
	if !ok || id.IsZero() {
		return "", domainerr.TenantMissing()
	}
	return id, nil
}

// RequireUUID es conveniencia para ponti/pymes: equivalente a
// `Require(ctx).UUID()` con manejo de errores agrupado.
func RequireUUID(ctx context.Context) (uuid.UUID, error) {
	id, err := Require(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	u, err := id.UUID()
	if err != nil {
		return uuid.Nil, domainerr.Validation("tenant id is not a valid uuid")
	}
	return u, nil
}

// --- Strict mode ---

var (
	strictOverrideMu  sync.RWMutex
	strictOverrideSet bool
	strictOverrideVal bool
)

// StrictModeEnabled retorna true si strict mode está activo. Precedencia:
//  1. Override programático (vía SetStrictMode) si fue seteado.
//  2. Env var `TENANT_STRICT_MODE` (1/true/yes/y/on) — leída en cada llamada
//     para compatibilidad con tests que usan `t.Setenv`.
//
// Sin cache: el costo de un `os.Getenv` por query es trivial (~ns) y evita
// edge cases de stale state.
func StrictModeEnabled() bool {
	strictOverrideMu.RLock()
	defer strictOverrideMu.RUnlock()
	if strictOverrideSet {
		return strictOverrideVal
	}
	return parseBool(os.Getenv("TENANT_STRICT_MODE"))
}

// SetStrictMode fuerza el valor de strict mode ignorando el env. Pensado
// para tests que quieren control determinístico sin t.Setenv. Llamar con
// `ResetStrictMode()` (o nuevo Set) para limpiar el override.
// Concurrency-safe.
func SetStrictMode(enabled bool) {
	strictOverrideMu.Lock()
	defer strictOverrideMu.Unlock()
	strictOverrideSet = true
	strictOverrideVal = enabled
}

// ResetStrictMode borra el override programático; vuelve a leer env.
func ResetStrictMode() {
	strictOverrideMu.Lock()
	defer strictOverrideMu.Unlock()
	strictOverrideSet = false
	strictOverrideVal = false
}

func parseBool(raw string) bool {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}

func coerce(v any) ID {
	switch t := v.(type) {
	case ID:
		return t
	case string:
		return FromString(t)
	case uuid.UUID:
		return FromUUID(t)
	case interface{ String() string }:
		return FromString(t.String())
	default:
		return ""
	}
}
