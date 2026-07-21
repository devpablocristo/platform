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

func TestDecimalRoundingRejectsInvalidModeOnEveryPath(t *testing.T) {
	invalid := RoundMode(99)
	tests := []struct {
		name  string
		input string
		scale int32
	}{
		{name: "no-op", input: "1.23", scale: 2},
		{name: "scale increase", input: "1.23", scale: 3},
		{name: "non-tie", input: "1.236", scale: 2},
		{name: "tie", input: "1.235", scale: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := MustParseDecimal(tt.input).Round(tt.scale, invalid); !errors.Is(err, ErrInvalidRoundMode) {
				t.Fatalf("Round(%d, %d) error = %v", tt.scale, invalid, err)
			}
		})
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

func TestDecimalScanRejectsSQLNull(t *testing.T) {
	scanned := MustParseDecimal("42.9")
	if err := scanned.Scan(nil); !errors.Is(err, ErrNullDecimal) {
		t.Fatalf("Scan(nil) error = %v", err)
	}
	if got := scanned.String(); got != "42.9" {
		t.Fatalf("Scan(nil) mutated receiver to %s", got)
	}
}
