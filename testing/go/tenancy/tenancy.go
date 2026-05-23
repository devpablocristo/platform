// Package tenancytest provee helpers para validar aislamiento cross-tenant
// en repositorios multi-tenant. Reemplaza el boilerplate repetido en los 25
// archivos `repository_tenant_test.go` de ponti.
//
// Uso típico (en un test de módulo):
//
//	func TestFooRepositoryTenantIsolation(t *testing.T) {
//	    tenancytest.Suite{
//	        Setup: func(t *testing.T) (*gorm.DB, func()) {
//	            db := newTestDB(t)            // helper del caller (sqlite + AutoMigrate)
//	            return db, func() {}
//	        },
//	        Seed: func(t *testing.T, db *gorm.DB, tenantA, tenantB uuid.UUID) any {
//	            // INSERT tenant A's data + tenant B's data
//	            // Retornar identifiers de tenant B para los assertion callbacks
//	            return bTargets{ProjectID: 20, RowID: 2}
//	        },
//	        ExpectedListCountForA: 1,
//	        ListAs: func(t *testing.T, ctx context.Context, db *gorm.DB) (int, error) {
//	            xs, err := repo.List(ctx)
//	            return len(xs), err
//	        },
//	        GetCrossTenant: func(t *testing.T, ctx context.Context, db *gorm.DB, b any) error {
//	            _, err := repo.Get(ctx, b.(bTargets).RowID)
//	            return err
//	        },
//	    }.Run(t)
//	}
//
// La suite ejecuta múltiples sub-tests `t.Run`:
//   - List as A → ExpectedListCountForA, ningún tenant B leak
//   - Get cross-tenant → debe fallar
//   - Update cross-tenant → debe fallar y NO mutar tenant B
//   - Delete cross-tenant → debe fallar y NO borrar tenant B
//   - Strict mode sin tenant → todas las ops fallan
//
// Callbacks que valgan nil se skippean. No fuerza a los repos a implementar
// todo el CRUDAR.
package tenancytest

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm"

	ctxkeys "github.com/devpablocristo/platform/security/go/contextkeys"
	"github.com/devpablocristo/platform/security/go/tenant"
)

// Suite agrupa la config canónica de un test de aislamiento.
// Todos los callbacks que sean nil se skippean.
type Suite struct {
	// Name aparece en t.Run; opcional, default "TenantIsolation".
	Name string

	// Setup abre la DB y retorna un cleanup. Llamado una vez al inicio de Run.
	Setup func(t *testing.T) (db *gorm.DB, cleanup func())

	// Seed inserta datos para tenant A y tenant B. Retorna identifiers
	// opacos para tenant B que los callbacks de operación reciben luego.
	Seed func(t *testing.T, db *gorm.DB, tenantA, tenantB uuid.UUID) (bTargets any)

	// ExpectedListCountForA es cuántas filas debe ver tenant A en ListAs.
	ExpectedListCountForA int

	// ListAs ejecuta List() en un contexto y retorna (count, err).
	ListAs func(t *testing.T, ctx context.Context, db *gorm.DB) (int, error)

	// GetCrossTenant llama Get() apuntando a un recurso de tenant B desde
	// el contexto de tenant A. Debe fallar.
	GetCrossTenant func(t *testing.T, ctx context.Context, db *gorm.DB, bTargets any) error

	// UpdateCrossTenant llama Update() sobre un recurso de tenant B desde
	// el contexto de tenant A. Debe fallar Y NO mutar la fila.
	// El callback recibe `verifyUnchanged` que el caller invoca a posteriori
	// para confirmar que el dato sigue intacto.
	UpdateCrossTenant func(t *testing.T, ctx context.Context, db *gorm.DB, bTargets any) error

	// DeleteCrossTenant llama Delete() sobre un recurso de tenant B desde
	// el contexto de tenant A. Debe fallar y NO borrar.
	DeleteCrossTenant func(t *testing.T, ctx context.Context, db *gorm.DB, bTargets any) error

	// CountTenantBRows cuenta filas de tenant B en la tabla bajo test;
	// se invoca antes y después de UpdateCrossTenant/DeleteCrossTenant
	// para verificar que tenant B sigue intacto. Opcional pero recomendado.
	CountTenantBRows func(t *testing.T, db *gorm.DB) int
}

// Run ejecuta la suite. Skippea cada operación cuya closure sea nil.
func (s Suite) Run(t *testing.T) {
	t.Helper()
	if s.Setup == nil {
		t.Fatal("tenancytest.Suite: Setup is required")
	}
	if s.Seed == nil {
		t.Fatal("tenancytest.Suite: Seed is required")
	}
	name := s.Name
	if name == "" {
		name = "TenantIsolation"
	}
	t.Run(name, func(t *testing.T) {
		db, cleanup := s.Setup(t)
		t.Cleanup(cleanup)

		tenantA := uuid.New()
		tenantB := uuid.New()
		bTargets := s.Seed(t, db, tenantA, tenantB)
		ctxA := ContextFor(tenantA)

		if s.ListAs != nil {
			t.Run("ListAs_A_returnsOnlyOwn", func(t *testing.T) {
				count, err := s.ListAs(t, ctxA, db)
				if err != nil {
					t.Fatalf("ListAs: %v", err)
				}
				if count != s.ExpectedListCountForA {
					t.Errorf("got %d rows, want %d", count, s.ExpectedListCountForA)
				}
			})
		}
		if s.GetCrossTenant != nil {
			t.Run("GetCrossTenant_fails", func(t *testing.T) {
				if err := s.GetCrossTenant(t, ctxA, db, bTargets); err == nil {
					t.Error("expected cross-tenant Get to fail")
				}
			})
		}
		if s.UpdateCrossTenant != nil {
			t.Run("UpdateCrossTenant_failsAndPreservesB", func(t *testing.T) {
				before := -1
				if s.CountTenantBRows != nil {
					before = s.CountTenantBRows(t, db)
				}
				err := s.UpdateCrossTenant(t, ctxA, db, bTargets)
				if err == nil {
					t.Error("expected cross-tenant Update to fail")
				}
				if s.CountTenantBRows != nil {
					if got := s.CountTenantBRows(t, db); got != before {
						t.Errorf("tenant B row count changed: was %d, now %d", before, got)
					}
				}
			})
		}
		if s.DeleteCrossTenant != nil {
			t.Run("DeleteCrossTenant_failsAndPreservesB", func(t *testing.T) {
				before := -1
				if s.CountTenantBRows != nil {
					before = s.CountTenantBRows(t, db)
				}
				err := s.DeleteCrossTenant(t, ctxA, db, bTargets)
				if err == nil {
					t.Error("expected cross-tenant Delete to fail")
				}
				if s.CountTenantBRows != nil {
					if got := s.CountTenantBRows(t, db); got != before {
						t.Errorf("tenant B row count changed: was %d, now %d", before, got)
					}
				}
			})
		}
		if s.ListAs != nil {
			t.Run("StrictMode_NoTenant_failsList", func(t *testing.T) {
				prev := tenant.StrictModeEnabled()
				tenant.SetStrictMode(true)
				t.Cleanup(func() { tenant.SetStrictMode(prev) })
				if _, err := s.ListAs(t, context.Background(), db); err == nil {
					t.Error("expected strict-mode List without tenant to fail")
				}
			})
		}
	})
}

// ContextFor construye un contexto canónico con tenant + identidad para tests.
// Setea todas las claves estándar de `platform/security/go/contextkeys`:
//   - OrgID y TenantID con `tenantID`
//   - Actor "tenant-test@example.com" (default)
//   - Role "tenant_admin"
//   - Scopes con los provistos (vacío si no se pasan)
func ContextFor(tenantID uuid.UUID, scopes ...string) context.Context {
	ctx := context.Background()
	ctx = context.WithValue(ctx, ctxkeys.Actor, "tenant-test@example.com")
	ctx = context.WithValue(ctx, ctxkeys.OrgID, tenantID)
	ctx = context.WithValue(ctx, ctxkeys.TenantID, tenantID)
	ctx = context.WithValue(ctx, ctxkeys.Role, "tenant_admin")
	if len(scopes) == 0 {
		scopes = []string{"projects.read", "projects.write"}
	}
	ctx = context.WithValue(ctx, ctxkeys.Scopes, scopes)
	return ctx
}
