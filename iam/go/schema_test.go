package iam

import (
	"io/fs"
	"strings"
	"testing"
)

func TestMigrationProfileOwnsIAMSchema(t *testing.T) {
	t.Parallel()

	profile := MigrationProfile()
	if profile.Scope != MigrationScope || profile.Dir != MigrationsDir {
		t.Fatalf("unexpected migration profile: %#v", profile)
	}
	body, err := fs.ReadFile(profile.Migrations, MigrationsDir+"/0001_iam_core.sql")
	if err != nil {
		t.Fatalf("read embedded migration: %v", err)
	}
	sql := string(body)
	for _, required := range []string{
		"CREATE SCHEMA IF NOT EXISTS iam",
		"iam.organizations",
		"iam.users",
		"iam.memberships",
		"iam.invitations",
		"iam.webhook_events",
	} {
		if !strings.Contains(sql, required) {
			t.Errorf("migration does not contain %q", required)
		}
	}
	for _, forbidden := range []string{"billing", "owner", "clerk", "pymes"} {
		if strings.Contains(strings.ToLower(sql), forbidden) {
			t.Errorf("migration contains product/provider concept %q", forbidden)
		}
	}
}
