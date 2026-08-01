package scheduling

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPublishedCoreHasNoRuntimeOrPersistenceAdapters(t *testing.T) {
	t.Parallel()

	for _, path := range []string{"httpgin", "publichttpgin", "migrations", "repository", "repository.go", "seeds"} {
		if _, err := os.Stat(path); err == nil {
			t.Errorf("published scheduling core must not contain %s", path)
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat %s: %v", path, err)
		}
	}

	forbidden := []string{
		"github.com/gin-gonic/gin",
		"gorm.io/",
		"github.com/jackc/pgx",
		"database/sql",
		"OrganizationID",
		`json:"org_id"`,
		"ListBookingsByPhone",
		"digitsOnly",
	}
	err := filepath.WalkDir(".", func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(path) != ".go" {
			return nil
		}
		if path == "architecture_test.go" {
			return nil
		}
		source, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, dependency := range forbidden {
			if strings.Contains(string(source), dependency) {
				t.Errorf("%s contains forbidden runtime dependency %q", path, dependency)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestGoModuleUsesOnlyPublishedDependencies(t *testing.T) {
	t.Parallel()

	source, err := os.ReadFile("go.mod")
	if err != nil {
		t.Fatal(err)
	}
	manifest := string(source)
	for _, forbidden := range []string{"replace ", "v0.0.0-00010101000000-000000000000", "gin-gonic", "gorm.io"} {
		if strings.Contains(manifest, forbidden) {
			t.Errorf("go.mod contains forbidden dependency marker %q", forbidden)
		}
	}
}
