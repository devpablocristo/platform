package civil

import (
	"encoding/json"
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	cases := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"", "", false},
		{"2024-03-15", "2024-03-15", false},
		{"  2024-03-15  ", "2024-03-15", false},
		{"2024-01-15T10:30:00Z", "2024-01-15", false}, // RFC3339 tolerado
		{"2024-01-15 10:30:00", "2024-01-15", false},  // separador espacio
		{"15/03/2024", "", true},
		{"2024-13-40", "", true},
		{"not-a-date", "", true},
	}
	for _, c := range cases {
		d, err := Parse(c.in)
		if c.wantErr {
			if err == nil {
				t.Errorf("Parse(%q): esperaba error", c.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("Parse(%q): error inesperado %v", c.in, err)
			continue
		}
		if d.String() != c.want {
			t.Errorf("Parse(%q) = %q, want %q", c.in, d.String(), c.want)
		}
	}
}

func TestCompare(t *testing.T) {
	a := MustParse("2024-01-01")
	b := MustParse("2024-12-31")
	zero := Date{}
	if !a.Before(b) || !b.After(a) {
		t.Fatal("orden a<b mal")
	}
	if a.Before(zero) || zero.After(a) {
		t.Fatal("comparar contra zero debe ser false")
	}
	if !a.Equal(MustParse("2024-01-01")) {
		t.Fatal("Equal mal")
	}
	if !zero.IsZero() || !(Date{}).IsZero() {
		t.Fatal("IsZero mal")
	}
}

func TestJSONRoundTrip(t *testing.T) {
	type wrap struct {
		D Date `json:"d"`
	}
	in := wrap{D: MustParse("2024-06-30")}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `{"d":"2024-06-30"}` {
		t.Fatalf("marshal = %s", b)
	}
	var out wrap
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if !out.D.Equal(in.D) {
		t.Fatalf("round-trip mismatch: %v vs %v", out.D, in.D)
	}
	// zero → ""
	zb, _ := json.Marshal(wrap{})
	if string(zb) != `{"d":""}` {
		t.Fatalf("zero marshal = %s", zb)
	}
}

func TestSQLValueScan(t *testing.T) {
	d := MustParse("2024-02-29")
	v, err := d.Value()
	if err != nil {
		t.Fatal(err)
	}
	if v != "2024-02-29" {
		t.Fatalf("Value = %v", v)
	}
	// zero → NULL
	zv, _ := Date{}.Value()
	if zv != nil {
		t.Fatalf("zero Value = %v, want nil", zv)
	}
	// Scan desde string, []byte, time.Time, nil
	var got Date
	for _, src := range []any{"2024-02-29", []byte("2024-02-29"), time.Date(2024, 2, 29, 12, 0, 0, 0, time.UTC)} {
		if err := got.Scan(src); err != nil {
			t.Fatalf("Scan(%T) error %v", src, err)
		}
		if got.String() != "2024-02-29" {
			t.Fatalf("Scan(%T) = %q", src, got.String())
		}
	}
	if err := got.Scan(nil); err != nil || !got.IsZero() {
		t.Fatalf("Scan(nil) → %q err %v", got.String(), err)
	}
}
