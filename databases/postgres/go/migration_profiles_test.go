package postgres

import (
	"context"
	"strings"
	"testing"
	"testing/fstest"
)

func TestMigrateProfilesRunsInOrderAndIsIdempotent(t *testing.T) {
	t.Parallel()

	db := newFakeMigrationDB()
	profiles := []MigrationProfile{
		{
			Scope: "first",
			Migrations: fstest.MapFS{
				"migrations/0001_first.sql": {Data: []byte("CREATE TABLE first_table")},
			},
			Dir: "migrations",
		},
		{
			Scope: "second",
			Migrations: fstest.MapFS{
				"migrations/0001_second.sql": {Data: []byte("CREATE TABLE second_table")},
			},
			Dir: "migrations",
		},
	}

	for run := 0; run < 2; run++ {
		if err := MigrateProfiles(context.Background(), db, profiles...); err != nil {
			t.Fatalf("MigrateProfiles run %d returned error: %v", run+1, err)
		}
	}

	if got := strings.Join(db.appliedSQL, "|"); got != "CREATE TABLE first_table|CREATE TABLE second_table" {
		t.Fatalf("unexpected applied SQL order: %s", got)
	}
	if len(db.ensureSchemaCalls) != 2 {
		t.Fatalf("expected schema table once per run, got %d", len(db.ensureSchemaCalls))
	}
}

func TestMigrateProfilesDefaultsToFilesystemRoot(t *testing.T) {
	t.Parallel()

	db := newFakeMigrationDB()
	err := MigrateProfiles(context.Background(), db, MigrationProfile{
		Scope: "root",
		Migrations: fstest.MapFS{
			"0001_root.sql": {Data: []byte("CREATE TABLE root_table")},
		},
	})
	if err != nil {
		t.Fatalf("MigrateProfiles returned error: %v", err)
	}
	if got := strings.Join(db.appliedSQL, "|"); got != "CREATE TABLE root_table" {
		t.Fatalf("unexpected applied SQL: %s", got)
	}
}

func TestMigrateProfilesPreflightRejectsInvalidProfilesBeforeWrites(t *testing.T) {
	t.Parallel()

	valid := MigrationProfile{
		Scope: "valid",
		Migrations: fstest.MapFS{
			"0001_valid.sql": {Data: []byte("CREATE TABLE valid_table")},
		},
	}
	tests := []struct {
		name     string
		profiles []MigrationProfile
		want     string
	}{
		{
			name:     "empty scope",
			profiles: []MigrationProfile{{Migrations: fstest.MapFS{"0001.sql": {Data: []byte("SELECT 1")}}}},
			want:     "scope required",
		},
		{
			name:     "nil filesystem",
			profiles: []MigrationProfile{{Scope: "missing"}},
			want:     "migrations required",
		},
		{
			name:     "duplicate scope",
			profiles: []MigrationProfile{valid, {Scope: " valid ", Migrations: valid.Migrations}},
			want:     "duplicate scope",
		},
		{
			name: "missing directory in later profile",
			profiles: []MigrationProfile{
				valid,
				{Scope: "broken", Migrations: fstest.MapFS{"0001.sql": {Data: []byte("SELECT 1")}}, Dir: "missing"},
			},
			want: `prepare migration profile "broken"`,
		},
		{
			name: "empty SQL in later profile",
			profiles: []MigrationProfile{
				valid,
				{Scope: "empty", Migrations: fstest.MapFS{"0001.sql": {Data: []byte("   ")}}},
			},
			want: `prepare migration profile "empty"`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			db := newFakeMigrationDB()
			err := MigrateProfiles(context.Background(), db, test.profiles...)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected error containing %q, got %v", test.want, err)
			}
			if len(db.ensureSchemaCalls) != 0 || len(db.appliedSQL) != 0 || db.beginCalls != 0 {
				t.Fatalf("preflight wrote to database: %#v", db)
			}
		})
	}
}

func TestMigrateProfilesStopsAfterSQLFailure(t *testing.T) {
	t.Parallel()

	db := newFakeMigrationDB()
	db.failSQL = "CREATE TABLE broken_table"
	err := MigrateProfiles(context.Background(), db,
		MigrationProfile{
			Scope:      "first",
			Migrations: fstest.MapFS{"0001.sql": {Data: []byte("CREATE TABLE first_table")}},
		},
		MigrationProfile{
			Scope:      "second",
			Migrations: fstest.MapFS{"0001.sql": {Data: []byte("CREATE TABLE broken_table")}},
		},
		MigrationProfile{
			Scope:      "third",
			Migrations: fstest.MapFS{"0001.sql": {Data: []byte("CREATE TABLE third_table")}},
		},
	)
	if err == nil || !strings.Contains(err.Error(), "second/0001.sql") {
		t.Fatalf("expected scoped migration error, got %v", err)
	}
	if got := strings.Join(db.appliedSQL, "|"); got != "CREATE TABLE first_table" {
		t.Fatalf("unexpected applied SQL after failure: %s", got)
	}
}

func TestMigrateProfilesWithNoProfilesIsNoOp(t *testing.T) {
	t.Parallel()

	if err := MigrateProfiles(context.Background(), nil); err != nil {
		t.Fatalf("MigrateProfiles with no profiles returned error: %v", err)
	}
}
