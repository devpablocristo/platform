package tabular

import (
	"testing"
)

func TestParseCSV(t *testing.T) {
	t.Parallel()

	sheet, err := ParseCSV([]byte("\xEF\xBB\xBFname,email\nJuan,juan@example.com\n"))
	if err != nil {
		t.Fatalf("ParseCSV returned err: %v", err)
	}
	if got, want := sheet.Headers[0], "name"; got != want {
		t.Fatalf("unexpected header: got=%q want=%q", got, want)
	}
	if got, want := sheet.Rows[0][1], "juan@example.com"; got != want {
		t.Fatalf("unexpected value: got=%q want=%q", got, want)
	}
}

func TestParseXLSX(t *testing.T) {
	t.Parallel()

	body, err := XLSX([]string{"name", "email"}, [][]string{{"Juan", "juan@example.com"}})
	if err != nil {
		t.Fatalf("XLSX returned err: %v", err)
	}

	sheet, err := ParseXLSX(body, "")
	if err != nil {
		t.Fatalf("ParseXLSX returned err: %v", err)
	}
	if got, want := sheet.Headers[1], "email"; got != want {
		t.Fatalf("unexpected header: got=%q want=%q", got, want)
	}
	if got, want := sheet.Rows[0][0], "Juan"; got != want {
		t.Fatalf("unexpected value: got=%q want=%q", got, want)
	}
}
