package tenancytest_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	ctxkeys "github.com/devpablocristo/platform/security/go/contextkeys"
	tenancytest "github.com/devpablocristo/platform/testing/go/tenancy"
)

type item struct {
	ID       uint
	Name     string
	TenantID string `gorm:"column:tenant_id"`
}

func (item) TableName() string { return "items" }

type bTargets struct {
	ID uint
}

func setup(t *testing.T) (*gorm.DB, func()) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&item{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db, func() {}
}

func TestSuite_HappyPath_AllAssertionsPass(t *testing.T) {
	tenancytest.Suite{
		Name:                  "happy",
		Setup:                 setup,
		ExpectedListCountForA: 2,
		Seed: func(t *testing.T, db *gorm.DB, tenantA, tenantB uuid.UUID) any {
			rows := []item{
				{Name: "a-1", TenantID: tenantA.String()},
				{Name: "a-2", TenantID: tenantA.String()},
				{Name: "b-1", TenantID: tenantB.String()},
			}
			if err := db.Create(&rows).Error; err != nil {
				t.Fatalf("seed: %v", err)
			}
			// Identificador de tenant B para los callbacks
			var b item
			if err := db.First(&b, "tenant_id = ?", tenantB.String()).Error; err != nil {
				t.Fatalf("lookup b: %v", err)
			}
			return bTargets{ID: b.ID}
		},
		ListAs: func(t *testing.T, ctx context.Context, db *gorm.DB) (int, error) {
			tenantID, _ := ctx.Value(ctxkeys.OrgID).(uuid.UUID)
			if tenantID == uuid.Nil {
				return 0, errors.New("no tenant in ctx")
			}
			var xs []item
			err := db.Where("tenant_id = ?", tenantID.String()).Find(&xs).Error
			return len(xs), err
		},
		GetCrossTenant: func(t *testing.T, ctx context.Context, db *gorm.DB, bt any) error {
			tenantID, _ := ctx.Value(ctxkeys.OrgID).(uuid.UUID)
			var x item
			return db.Where("id = ? AND tenant_id = ?", bt.(bTargets).ID, tenantID.String()).First(&x).Error
		},
		UpdateCrossTenant: func(t *testing.T, ctx context.Context, db *gorm.DB, bt any) error {
			tenantID, _ := ctx.Value(ctxkeys.OrgID).(uuid.UUID)
			res := db.Model(&item{}).
				Where("id = ? AND tenant_id = ?", bt.(bTargets).ID, tenantID.String()).
				Update("name", "MUTATED")
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected == 0 {
				return errors.New("no rows affected")
			}
			return nil
		},
		DeleteCrossTenant: func(t *testing.T, ctx context.Context, db *gorm.DB, bt any) error {
			tenantID, _ := ctx.Value(ctxkeys.OrgID).(uuid.UUID)
			res := db.Where("id = ? AND tenant_id = ?", bt.(bTargets).ID, tenantID.String()).
				Delete(&item{})
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected == 0 {
				return errors.New("no rows affected")
			}
			return nil
		},
		CountTenantBRows: func(t *testing.T, db *gorm.DB) int {
			var rows []item
			if err := db.Where("name LIKE ?", "b-%").Find(&rows).Error; err != nil {
				t.Fatalf("count B: %v", err)
			}
			return len(rows)
		},
	}.Run(t)
}

func TestContextFor_PopulatesAllKeys(t *testing.T) {
	t.Parallel()
	tenantID := uuid.New()
	ctx := tenancytest.ContextFor(tenantID, "x.read", "x.write")
	if got, _ := ctx.Value(ctxkeys.OrgID).(uuid.UUID); got != tenantID {
		t.Errorf("OrgID=%v", got)
	}
	if got, _ := ctx.Value(ctxkeys.TenantID).(uuid.UUID); got != tenantID {
		t.Errorf("TenantID=%v", got)
	}
	if got, _ := ctx.Value(ctxkeys.Actor).(string); got == "" {
		t.Error("Actor empty")
	}
	if got, _ := ctx.Value(ctxkeys.Scopes).([]string); len(got) != 2 {
		t.Errorf("Scopes=%v", got)
	}
}

func TestContextFor_DefaultScopes(t *testing.T) {
	t.Parallel()
	ctx := tenancytest.ContextFor(uuid.New())
	got, _ := ctx.Value(ctxkeys.Scopes).([]string)
	if len(got) == 0 {
		t.Error("expected default scopes")
	}
}
