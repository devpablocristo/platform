package artifact

import (
	"testing"
	"time"
)

func TestNewNormalizesAsset(t *testing.T) {
	t.Parallel()

	item := New("sales_export", FormatCSV, []byte("body"), map[string]string{"tenant": "acme"})
	if item.Name != "sales_export.csv" {
		t.Fatalf("unexpected name: %q", item.Name)
	}
	if item.ContentType != "text/csv; charset=utf-8" {
		t.Fatalf("unexpected content type: %q", item.ContentType)
	}
	if item.Size() != 4 {
		t.Fatalf("unexpected size: %d", item.Size())
	}
	if item.Metadata["tenant"] != "acme" {
		t.Fatalf("unexpected metadata: %#v", item.Metadata)
	}
}

func TestBuildFilename(t *testing.T) {
	t.Parallel()

	got := BuildFilename([]string{"Sales Report", "ACME/Prod"}, FormatPDF, time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC))
	if got != "sales_report_acme_prod_2026-03-20.pdf" {
		t.Fatalf("unexpected filename: %q", got)
	}
}
