package clerk

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestMetadataPreservesNumbersAndNestedObjects(t *testing.T) {
	var metadata Metadata
	if err := json.Unmarshal([]byte(`{"attempt":9007199254740993,"nested":{"active":true}}`), &metadata); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if metadata["attempt"] != json.Number("9007199254740993") {
		t.Fatalf("number lost precision: %#v", metadata["attempt"])
	}
	nested, ok := metadata["nested"].(map[string]any)
	if !ok || nested["active"] != true {
		t.Fatalf("unexpected nested metadata %#v", metadata)
	}
}

func TestMetadataRejectsNonObjectJSON(t *testing.T) {
	for _, raw := range []string{`[]`, `"marker"`, `42`, `true`} {
		var metadata Metadata
		err := json.Unmarshal([]byte(raw), &metadata)
		if !errors.Is(err, ErrInvalidMetadata) {
			t.Fatalf("%s: expected ErrInvalidMetadata, got %v", raw, err)
		}
	}
}
