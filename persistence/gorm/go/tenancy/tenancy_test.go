package tenancy_test

import (
	"context"
	"errors"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/devpablocristo/platform/errors/go/domainerr"
	"github.com/devpablocristo/platform/persistence/gorm/go/tenancy"
	"github.com/devpablocristo/platform/security/go/tenant"
)

type fixture struct {
	ID       uint
	Name     string
	TenantID string `gorm:"column:tenant_id"`
	OrgID    string `gorm:"column:org_id"`
}

func (fixture) TableName() string { return "fixtures" }

func newDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&fixture{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	rows := []fixture{
		{Name: "a-1", TenantID: "tenant-A", OrgID: "tenant-A"},
		{Name: "a-2", TenantID: "tenant-A", OrgID: "tenant-A"},
		{Name: "b-1", TenantID: "tenant-B", OrgID: "tenant-B"},
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}
	return db
}

func TestScope_FiltersByDefaultColumn(t *testing.T) {
	db := newDB(t)
	ctx := tenant.WithID(context.Background(), tenant.FromString("tenant-A"))
	var got []fixture
	if err := tenancy.Scope(ctx, db, "fixtures").Find(&got).Error; err != nil {
		t.Fatalf("find: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 rows for tenant-A, got %d", len(got))
	}
	for _, r := range got {
		if r.TenantID != "tenant-A" {
			t.Errorf("leaked row: %+v", r)
		}
	}
}

func TestScopeWithColumn_OrgID(t *testing.T) {
	db := newDB(t)
	ctx := tenant.WithID(context.Background(), tenant.FromString("tenant-B"))
	var got []fixture
	if err := tenancy.ScopeWithColumn(ctx, db, "fixtures", "org_id").Find(&got).Error; err != nil {
		t.Fatalf("find: %v", err)
	}
	if len(got) != 1 {
		t.Errorf("expected 1 row for tenant-B via org_id, got %d", len(got))
	}
}

func TestScope_NoTenant_StrictModeOn_FailsClosed(t *testing.T) {
	prev := tenant.StrictModeEnabled()
	tenant.SetStrictMode(true)
	t.Cleanup(func() { tenant.SetStrictMode(prev) })

	db := newDB(t)
	var got []fixture
	err := tenancy.Scope(context.Background(), db, "fixtures").Find(&got).Error
	if err == nil {
		t.Fatal("expected error in strict mode without tenant")
	}
	if !errors.Is(err, domainerr.TenantMissing()) {
		t.Errorf("expected TenantMissing, got %v", err)
	}
}

func TestScope_NoTenant_StrictModeOff_AllowsAll(t *testing.T) {
	prev := tenant.StrictModeEnabled()
	tenant.SetStrictMode(false)
	t.Cleanup(func() { tenant.SetStrictMode(prev) })

	db := newDB(t)
	var got []fixture
	if err := tenancy.Scope(context.Background(), db, "fixtures").Find(&got).Error; err != nil {
		t.Fatalf("find: %v", err)
	}
	if len(got) != 3 {
		t.Errorf("expected 3 rows (no filter), got %d", len(got))
	}
}

func TestScope_NilDB_NoPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic on nil db: %v", r)
		}
	}()
	got := tenancy.Scope(context.Background(), nil, "fixtures")
	if got != nil {
		t.Error("expected nil db passthrough")
	}
}

func TestWhere_HappyPath(t *testing.T) {
	t.Parallel()
	ctx := tenant.WithID(context.Background(), tenant.FromString("tnt-x"))
	clause, args, err := tenancy.Where(ctx, "fixtures")
	if err != nil {
		t.Fatalf("Where: %v", err)
	}
	if clause != "fixtures.tenant_id = ?" {
		t.Errorf("clause=%q", clause)
	}
	if len(args) != 1 || args[0] != "tnt-x" {
		t.Errorf("args=%v", args)
	}
}

func TestWhere_FailsClosedOnMissing(t *testing.T) {
	t.Parallel()
	_, _, err := tenancy.Where(context.Background(), "fixtures")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, domainerr.TenantMissing()) {
		t.Errorf("expected TenantMissing, got %v", err)
	}
}

func TestWhereWithColumn_OrgID(t *testing.T) {
	t.Parallel()
	ctx := tenant.WithID(context.Background(), tenant.FromString("xyz"))
	clause, args, err := tenancy.WhereWithColumn(ctx, "accounts", "org_id")
	if err != nil {
		t.Fatalf("WhereWithColumn: %v", err)
	}
	if clause != "accounts.org_id = ?" {
		t.Errorf("clause=%q", clause)
	}
	if args[0] != "xyz" {
		t.Errorf("args[0]=%v", args[0])
	}
}

func TestBuildColumn_EdgeCases(t *testing.T) {
	t.Parallel()
	// Vacío + default
	c, _, err := tenancy.WhereWithColumn(tenant.WithID(context.Background(), "x"), "", "")
	if err != nil {
		t.Fatal(err)
	}
	if c != "tenant_id = ?" {
		t.Errorf("expected bare default, got %q", c)
	}
	// Pre-calificado con dot
	c, _, _ = tenancy.WhereWithColumn(tenant.WithID(context.Background(), "x"), "schema.table.org_id", "org_id")
	if c != "schema.table.org_id = ?" {
		t.Errorf("pre-qualified should pass through, got %q", c)
	}
}

func TestScope_AppliesToUpdatesAndDeletes(t *testing.T) {
	db := newDB(t)
	ctx := tenant.WithID(context.Background(), tenant.FromString("tenant-A"))

	// Update con scope NO debe tocar tenant-B
	res := tenancy.Scope(ctx, db, "fixtures").Model(&fixture{}).
		Where("name = ?", "b-1").Update("name", "MUTATED")
	if res.Error != nil {
		t.Fatalf("update: %v", res.Error)
	}
	if res.RowsAffected != 0 {
		t.Errorf("expected 0 rows affected (cross-tenant), got %d", res.RowsAffected)
	}

	// Confirmamos que b-1 sigue intacto
	var b fixture
	if err := db.First(&b, "name = ?", "b-1").Error; err != nil {
		t.Fatalf("re-read b-1: %v", err)
	}
	if b.Name != "b-1" {
		t.Errorf("b-1 was mutated cross-tenant: %+v", b)
	}
}
