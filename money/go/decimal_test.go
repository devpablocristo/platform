package money

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"testing"
)

func TestDecimalParseStringAndJSON(t *testing.T) {
	value := MustParseDecimal("00123.4500")
	if got := value.String(); got != "123.45" {
		t.Fatalf("String() = %q", got)
	}

	body, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != `"123.45"` {
		t.Fatalf("MarshalJSON() = %s", body)
	}

	var decoded Decimal
	if err := json.Unmarshal([]byte(`"123.45"`), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Cmp(value) != 0 {
		t.Fatalf("decoded %s != %s", decoded.String(), value.String())
	}

	if err := json.Unmarshal([]byte(`123.45`), &decoded); err == nil {
		t.Fatal("expected JSON numbers to be rejected")
	}
}

func TestDecimalRejectsInvalidInput(t *testing.T) {
	invalid := []string{"", ".", "1.2.3", "1e2", "abc"}
	for _, input := range invalid {
		if _, err := ParseDecimal(input); !errors.Is(err, ErrInvalidDecimal) {
			t.Fatalf("ParseDecimal(%q) error = %v", input, err)
		}
	}
}

func TestDecimalArithmetic(t *testing.T) {
	left := MustParseDecimal("10.25")
	right := MustParseDecimal("0.755")

	if got := left.Add(right).String(); got != "11.005" {
		t.Fatalf("Add() = %s", got)
	}
	if got := left.Sub(right).String(); got != "9.495" {
		t.Fatalf("Sub() = %s", got)
	}
	if got := MustParseDecimal("2.50").Mul(MustParseDecimal("4")).String(); got != "10" {
		t.Fatalf("Mul() = %s", got)
	}
}

func TestDecimalRounding(t *testing.T) {
	tests := []struct {
		input string
		mode  RoundMode
		want  string
	}{
		{"1.235", RoundHalfUp, "1.24"},
		{"1.225", RoundHalfEven, "1.22"},
		{"1.235", RoundHalfEven, "1.24"},
		{"-1.235", RoundHalfUp, "-1.24"},
		{"1.239", RoundDown, "1.23"},
	}

	for _, tt := range tests {
		got, err := MustParseDecimal(tt.input).Round(2, tt.mode)
		if err != nil {
			t.Fatal(err)
		}
		if got.String() != tt.want {
			t.Fatalf("%s Round(%d) = %s, want %s", tt.input, tt.mode, got.String(), tt.want)
		}
	}
}

func TestDecimalSQLInterfaces(t *testing.T) {
	value := MustParseDecimal("42.90")
	driverValue, err := value.Value()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := driverValue.(driver.Value); !ok {
		t.Fatalf("Value() did not return driver.Value: %T", driverValue)
	}
	if driverValue != "42.9" {
		t.Fatalf("Value() = %v", driverValue)
	}

	var scanned Decimal
	if err := scanned.Scan([]byte("42.900")); err != nil {
		t.Fatal(err)
	}
	if scanned.String() != "42.9" {
		t.Fatalf("Scan() = %s", scanned.String())
	}
	if err := scanned.Scan(42.9); !errors.Is(err, ErrUnsupportedScan) {
		t.Fatalf("expected unsupported scan, got %v", err)
	}
}
