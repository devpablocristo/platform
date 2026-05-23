// Package tenancy provee helpers GORM para aplicar scope tenant a queries
// de forma uniforme entre ponti (column `tenant_id`) y pymes (column `org_id`).
//
// Diseño:
//   - `Scope(ctx, db, "table")` agrega `table.tenant_id = ?` con el tenant del
//     contexto (lift exacto de ponti `authz.MaybeTenantScope`).
//   - `ScopeWithColumn(ctx, db, "table", "org_id")` permite el override de
//     columna (pymes usa `org_id`).
//   - Fail-closed: si el contexto no trae tenant Y strict mode está on,
//     agrega `domainerr.TenantMissing` al `*gorm.DB` para que `.Find()`
//     subsiguientes propaguen el error. Si strict mode está off, el query
//     se ejecuta sin filtro (status quo ponti pre-strict).
//   - `Where(ctx, "table")` retorna la cláusula + args como strings — para
//     callers que no pueden chainear sobre `*gorm.DB` (raw SQL, batch).
//
// Tenant id resuelto vía `platform/security/go/tenant.FromContext`.
// Strict mode resuelto vía `platform/security/go/tenant.StrictModeEnabled`.
package tenancy

import (
	"context"
	"strings"

	"gorm.io/gorm"

	"github.com/devpablocristo/platform/errors/go/domainerr"
	"github.com/devpablocristo/platform/security/go/tenant"
)

// DefaultColumn es la columna usada cuando no se especifica una.
// Match con la convención de ponti — pymes pasa `org_id` explícito.
const DefaultColumn = "tenant_id"

// Scope agrega `<table>.<DefaultColumn> = ?` al query, leyendo el tenant id
// del contexto. Comportamiento:
//
//   - db == nil → db tal cual (no panic).
//   - sin tenant en ctx + strict mode ON  → db con `.AddError(TenantMissing)`.
//   - sin tenant en ctx + strict mode OFF → db sin filtro (legacy behavior).
//   - con tenant → db.Where("<col> = ?", tenantID).
//
// `columnOrAlias`:
//   - vacío → `tenant_id` (sin prefijo de tabla; útil cuando hay 1 sola tabla).
//   - contiene `.` o `(` o ` ` → se usa tal cual (raw SQL).
//   - otherwise → `<value>.tenant_id` (asume nombre de tabla/alias).
func Scope(ctx context.Context, db *gorm.DB, columnOrAlias string) *gorm.DB {
	return ScopeWithColumn(ctx, db, columnOrAlias, DefaultColumn)
}

// ScopeWithColumn es la variante explícita: el caller indica nombre de columna.
// Pensado para pymes que usa `org_id` en lugar de `tenant_id`.
func ScopeWithColumn(ctx context.Context, db *gorm.DB, tableOrAlias, column string) *gorm.DB {
	if db == nil {
		return db
	}
	id, ok := tenant.FromContext(ctx)
	if !ok || id.IsZero() {
		if tenant.StrictModeEnabled() {
			err := domainerr.TenantMissing()
			if db.Config == nil {
				db.Error = err
			} else {
				_ = db.AddError(err)
			}
		}
		return db
	}
	col := buildColumn(tableOrAlias, column)
	return db.Where(col+" = ?", id.String())
}

// Where retorna la cláusula y args sin necesidad de un *gorm.DB. Útil para
// callers que arman SQL crudo (e.g., copy/insert batch). Falla cerrado:
// si no hay tenant en ctx, retorna `domainerr.TenantMissing` independiente
// del strict mode (porque sin contexto no se puede armar la cláusula).
func Where(ctx context.Context, columnOrAlias string) (string, []any, error) {
	return WhereWithColumn(ctx, columnOrAlias, DefaultColumn)
}

// WhereWithColumn igual que Where pero con columna explícita.
func WhereWithColumn(ctx context.Context, tableOrAlias, column string) (string, []any, error) {
	id, err := tenant.Require(ctx)
	if err != nil {
		return "", nil, err
	}
	col := buildColumn(tableOrAlias, column)
	return col + " = ?", []any{id.String()}, nil
}

// buildColumn aplica la heurística de prefijo de tabla:
//   - tableOrAlias vacío → `<column>` solo
//   - tableOrAlias contiene `.`, `(` o ` ` → usar tal cual (raw SQL del caller)
//   - otherwise → `<table>.<column>`
//
// Misma lógica que ponti `normalizeTenantColumn`, parametrizada por columna.
func buildColumn(tableOrAlias, column string) string {
	tableOrAlias = strings.TrimSpace(tableOrAlias)
	column = strings.TrimSpace(column)
	if column == "" {
		column = DefaultColumn
	}
	if tableOrAlias == "" {
		return column
	}
	if strings.ContainsAny(tableOrAlias, ".( ") {
		// Si ya viene calificado (`schema.tabla` o expresión), no mete prefijo.
		return tableOrAlias
	}
	return tableOrAlias + "." + column
}
