package tabular

import (
	"bytes"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestCSVIncludesBOMAndHeaders(t *testing.T) {
	t.Parallel()

	content, err := CSV([]string{"name", "email"}, [][]string{
		{"Juan", "juan@example.com"},
	})
	if err != nil {
		t.Fatalf("CSV returned err: %v", err)
	}
	if len(content) < 3 || content[0] != 0xEF || content[1] != 0xBB || content[2] != 0xBF {
		t.Fatal("expected UTF-8 BOM")
	}
	text := string(content[3:])
	if !strings.Contains(text, "name,email") {
		t.Fatalf("expected headers in csv, got %q", text)
	}
}

func TestXLSXProducesReadableWorkbook(t *testing.T) {
	t.Parallel()

	content, err := XLSX([]string{"name", "email"}, [][]string{
		{"Juan", "juan@example.com"},
	})
	if err != nil {
		t.Fatalf("XLSX returned err: %v", err)
	}

	f, err := excelize.OpenReader(bytes.NewReader(content))
	if err != nil {
		t.Fatalf("OpenReader returned err: %v", err)
	}
	defer func() { _ = f.Close() }()

	sheet := f.GetSheetName(f.GetActiveSheetIndex())
	value, err := f.GetCellValue(sheet, "A1")
	if err != nil {
		t.Fatalf("GetCellValue returned err: %v", err)
	}
	if got, want := value, "name"; got != want {
		t.Fatalf("unexpected header: got=%q want=%q", got, want)
	}
}

func TestCSVAsset(t *testing.T) {
	t.Parallel()

	item, err := CSVAsset("sales", []string{"name"}, [][]string{{"acme"}}, map[string]string{"tenant": "acme"})
	if err != nil {
		t.Fatalf("CSVAsset returned err: %v", err)
	}
	if item.Name != "sales.csv" {
		t.Fatalf("unexpected asset name: %q", item.Name)
	}
	if item.ContentType != CSVContentType {
		t.Fatalf("unexpected asset content type: %q", item.ContentType)
	}
	if !strings.Contains(string(item.Body), "acme") {
		t.Fatalf("unexpected asset body: %q", string(item.Body))
	}
}
