package money

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestCurrencyValidation(t *testing.T) {
	if _, err := ParseCurrency("ARS"); err != nil {
		t.Fatal(err)
	}
	for _, code := range []string{"ZZZ", "ars", "US", "USDT", "12A"} {
		if _, err := ParseCurrency(code); !errors.Is(err, ErrInvalidCurrency) {
			t.Fatalf("ParseCurrency(%q) error = %v", code, err)
		}
	}
}

func TestMoneyRoundRejectsInvalidMode(t *testing.T) {
	value, err := New(MustParseDecimal("10.01"), "ARS")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := value.Round(2, RoundMode(99)); !errors.Is(err, ErrInvalidRoundMode) {
		t.Fatalf("Round() error = %v", err)
	}
}

func TestMoneyJSON(t *testing.T) {
	value, err := New(MustParseDecimal("100.50"), "ARS")
	if err != nil {
		t.Fatal(err)
	}

	body, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != `{"amount":"100.5","currency":"ARS"}` {
		t.Fatalf("MarshalJSON() = %s", body)
	}

	var decoded Money
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Currency != "ARS" || decoded.Amount.String() != "100.5" {
		t.Fatalf("decoded = %+v", decoded)
	}
}

func TestMoneyCurrencyMismatch(t *testing.T) {
	ars, _ := New(MustParseDecimal("10"), "ARS")
	usd, _ := New(MustParseDecimal("10"), "USD")
	if _, err := ars.Add(usd); !errors.Is(err, ErrCurrencyMismatch) {
		t.Fatalf("expected currency mismatch, got %v", err)
	}
}

func TestMoneyAddRound(t *testing.T) {
	left, _ := New(MustParseDecimal("10.005"), "ARS")
	right, _ := New(MustParseDecimal("0.005"), "ARS")
	total, err := left.Add(right)
	if err != nil {
		t.Fatal(err)
	}
	rounded, err := total.Round(2, RoundHalfUp)
	if err != nil {
		t.Fatal(err)
	}
	if rounded.Amount.String() != "10.01" {
		t.Fatalf("rounded amount = %s", rounded.Amount.String())
	}
}
