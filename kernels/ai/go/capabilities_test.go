package ai

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCapabilityManifestValidExamples(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"valid_read_only.json", "valid_write_governed.json"} {
		name := name
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			manifest := loadCapabilityManifestFixture(t, name)
			if err := ValidateCapabilityManifest(manifest); err != nil {
				t.Fatalf("expected valid manifest %s: %v", name, err)
			}
		})
	}
}

func TestCapabilityManifestInvalidExamples(t *testing.T) {
	t.Parallel()

	tests := map[string][]string{
		"invalid_duplicate_tool.json":            {"duplicate tool name"},
		"invalid_write_missing_action_type.json": {"governance.action_type"},
		"invalid_read_side_effect.json":          {"side_effect=false"},
		"invalid_invalid_enum.json":              {"tenant_scope"},
		"invalid_secret_config.json":             {"runtime address", "forbidden configuration key"},
	}

	for name, wants := range tests {
		name, wants := name, wants
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			manifest := loadCapabilityManifestFixture(t, name)
			err := ValidateCapabilityManifest(manifest)
			if err == nil {
				t.Fatalf("expected validation error for %s", name)
			}
			if !containsAny(err.Error(), wants) {
				t.Fatalf("expected error containing one of %q, got %q", wants, err.Error())
			}
		})
	}
}

func TestCapabilityToolToToolPreservesLLMSurface(t *testing.T) {
	t.Parallel()

	manifest := loadCapabilityManifestFixture(t, "valid_write_governed.json")
	if err := ValidateCapabilityManifest(manifest); err != nil {
		t.Fatal(err)
	}

	tool := manifest.Tools[0].ToTool()
	if tool.Name != "pymes.sales.create" {
		t.Fatalf("unexpected tool name %q", tool.Name)
	}
	if tool.Description != manifest.Tools[0].Description {
		t.Fatalf("unexpected description %q", tool.Description)
	}
	if tool.Parameters["type"] != "object" {
		t.Fatalf("expected input_schema to be preserved as parameters, got %+v", tool.Parameters)
	}

	tools := manifest.ToolsForLLM()
	if len(tools) != 1 || tools[0].Name != tool.Name {
		t.Fatalf("unexpected converted tools: %+v", tools)
	}
}

func loadCapabilityManifestFixture(t *testing.T, name string) CapabilityManifest {
	t.Helper()

	data, err := os.ReadFile(filepath.Join("..", "contracts", "capabilities", "v1", "examples", name))
	if err != nil {
		t.Fatal(err)
	}
	var manifest CapabilityManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	return manifest
}

func containsAny(value string, needles []string) bool {
	for _, needle := range needles {
		if strings.Contains(value, needle) {
			return true
		}
	}
	return false
}
