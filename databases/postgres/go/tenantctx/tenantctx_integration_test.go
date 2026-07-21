package tenantctx

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/devpablocristo/platform/security/go/tenant"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestSetLocalScopesRLSAndDoesNotLeak(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("PLATFORM_POSTGRES_TEST_DSN"))
	if dsn == "" {
		if strings.EqualFold(os.Getenv("CI"), "true") {
			t.Fatal("PLATFORM_POSTGRES_TEST_DSN is required in CI")
		}
		t.Skip("PLATFORM_POSTGRES_TEST_DSN is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect pool: %v", err)
	}
	defer pool.Close()

	conn, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("acquire connection: %v", err)
	}
	defer conn.Release()

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	schemaName := "tenantctx_" + suffix
	roleName := "tenantctx_reader_" + suffix
	schema := pgx.Identifier{schemaName}.Sanitize()
	role := pgx.Identifier{roleName}.Sanitize()
	table := pgx.Identifier{schemaName, "records"}.Sanitize()

	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		_, _ = conn.Exec(cleanupCtx, "drop schema if exists "+schema+" cascade")
		_, _ = conn.Exec(cleanupCtx, "drop role if exists "+role)
	}()

	setup := []string{
		"create schema " + schema,
		"create role " + role + " nologin",
		"create table " + table + " (id text primary key, org_id uuid not null)",
		"alter table " + table + " enable row level security",
		"alter table " + table + " force row level security",
		"grant usage on schema " + schema + " to " + role,
		"grant select, insert on " + table + " to " + role,
		fmt.Sprintf(
			"create policy tenant_isolation on %s for all to %s using (org_id = nullif(current_setting('%s', true), '')::uuid) with check (org_id = nullif(current_setting('%s', true), '')::uuid)",
			table,
			role,
			Setting,
			Setting,
		),
	}
	for _, statement := range setup {
		if _, err := conn.Exec(ctx, statement); err != nil {
			t.Fatalf("setup %q: %v", statement, err)
		}
	}

	const (
		orgA = "11111111-1111-4111-8111-111111111111"
		orgB = "22222222-2222-4222-8222-222222222222"
	)
	if _, err := conn.Exec(ctx, "insert into "+table+" (id, org_id) values ($1, $2), ($3, $4)", "a", orgA, "b", orgB); err != nil {
		t.Fatalf("seed tenants: %v", err)
	}

	txA, err := conn.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tenant A: %v", err)
	}
	defer txA.Rollback(ctx) //nolint:errcheck
	if _, err := txA.Exec(ctx, "set local role "+role); err != nil {
		t.Fatalf("set tenant role: %v", err)
	}
	ctxA := tenant.WithID(ctx, tenant.FromString(orgA))
	if err := SetLocal(ctxA, txA); err != nil {
		t.Fatalf("SetLocal tenant A: %v", err)
	}

	var setting string
	if err := txA.QueryRow(ctx, "select current_setting($1, true)", Setting).Scan(&setting); err != nil {
		t.Fatalf("read tenant setting: %v", err)
	}
	if setting != orgA {
		t.Fatalf("setting = %q, want %q", setting, orgA)
	}

	rows, err := txA.Query(ctx, "select org_id::text from "+table+" order by org_id")
	if err != nil {
		t.Fatalf("query tenant rows: %v", err)
	}
	visible, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		t.Fatalf("collect tenant rows: %v", err)
	}
	if len(visible) != 1 || visible[0] != orgA {
		t.Fatalf("visible tenants = %v, want only %s", visible, orgA)
	}

	attack, err := txA.Begin(ctx)
	if err != nil {
		t.Fatalf("begin cross-tenant attack savepoint: %v", err)
	}
	if _, err := attack.Exec(ctx, "insert into "+table+" (id, org_id) values ($1, $2)", "attack", orgB); err == nil {
		_ = attack.Rollback(ctx)
		t.Fatal("cross-tenant insert unexpectedly succeeded")
	}
	if err := attack.Rollback(ctx); err != nil {
		t.Fatalf("rollback cross-tenant attack: %v", err)
	}
	if err := txA.Commit(ctx); err != nil {
		t.Fatalf("commit tenant A: %v", err)
	}

	assertNoTenantAccess := func(stage string) {
		t.Helper()
		tx, err := conn.Begin(ctx)
		if err != nil {
			t.Fatalf("begin after %s: %v", stage, err)
		}
		defer tx.Rollback(ctx) //nolint:errcheck
		if _, err := tx.Exec(ctx, "set local role "+role); err != nil {
			t.Fatalf("set role after %s: %v", stage, err)
		}
		var missing bool
		if err := tx.QueryRow(ctx, "select nullif(current_setting($1, true), '') is null", Setting).Scan(&missing); err != nil {
			t.Fatalf("read setting after %s: %v", stage, err)
		}
		if !missing {
			t.Fatalf("tenant setting leaked after %s", stage)
		}
		var count int
		if err := tx.QueryRow(ctx, "select count(*) from "+table).Scan(&count); err != nil {
			t.Fatalf("query without tenant after %s: %v", stage, err)
		}
		if count != 0 {
			t.Fatalf("visible rows without tenant after %s = %d, want 0", stage, count)
		}
		if err := tx.Commit(ctx); err != nil {
			t.Fatalf("commit no-tenant check after %s: %v", stage, err)
		}
	}

	assertNoTenantAccess("commit")

	txB, err := conn.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tenant B: %v", err)
	}
	if _, err := txB.Exec(ctx, "set local role "+role); err != nil {
		t.Fatalf("set tenant B role: %v", err)
	}
	ctxB := tenant.WithID(ctx, tenant.FromString(orgB))
	if err := SetLocal(ctxB, txB); err != nil {
		t.Fatalf("SetLocal tenant B: %v", err)
	}
	if err := txB.Rollback(ctx); err != nil {
		t.Fatalf("rollback tenant B: %v", err)
	}

	assertNoTenantAccess("rollback")
}
