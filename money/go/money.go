package money

import (
	"encoding/json"
	"errors"
	"fmt"

	"golang.org/x/text/currency"
)

var (
	ErrInvalidCurrency  = errors.New("money: invalid ISO 4217 currency")
	ErrCurrencyMismatch = errors.New("money: currency mismatch")
)

// Currency is an ISO 4217 alpha currency code.
type Currency string

// ParseCurrency validates and returns an ISO 4217 alpha currency code.
func ParseCurrency(code string) (Currency, error) {
	unit, err := currency.ParseISO(code)
	if err != nil || unit.String() != code {
		return "", fmt.Errorf("%w: %q", ErrInvalidCurrency, code)
	}
	return Currency(code), nil
}

func (c Currency) String() string {
	return string(c)
}

// MarshalJSON encodes a currency as a string.
func (c Currency) MarshalJSON() ([]byte, error) {
	if _, err := ParseCurrency(string(c)); err != nil {
		return nil, err
	}
	return json.Marshal(string(c))
}

// UnmarshalJSON decodes and validates a currency string.
func (c *Currency) UnmarshalJSON(data []byte) error {
	var raw string
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	parsed, err := ParseCurrency(raw)
	if err != nil {
		return err
	}
	*c = parsed
	return nil
}

// Money combines an exact amount with an ISO currency.
type Money struct {
	Amount   Decimal  `json:"amount"`
	Currency Currency `json:"currency"`
}

// New creates a Money value.
func New(amount Decimal, currency string) (Money, error) {
	code, err := ParseCurrency(currency)
	if err != nil {
		return Money{}, err
	}
	return Money{Amount: amount, Currency: code}, nil
}

// Add returns m + other when both currencies match.
func (m Money) Add(other Money) (Money, error) {
	if m.Currency != other.Currency {
		return Money{}, fmt.Errorf("%w: %s != %s", ErrCurrencyMismatch, m.Currency, other.Currency)
	}
	return Money{Amount: m.Amount.Add(other.Amount), Currency: m.Currency}, nil
}

// Sub returns m - other when both currencies match.
func (m Money) Sub(other Money) (Money, error) {
	if m.Currency != other.Currency {
		return Money{}, fmt.Errorf("%w: %s != %s", ErrCurrencyMismatch, m.Currency, other.Currency)
	}
	return Money{Amount: m.Amount.Sub(other.Amount), Currency: m.Currency}, nil
}

// Round returns a copy with amount rounded to scale.
func (m Money) Round(scale int32, mode RoundMode) (Money, error) {
	amount, err := m.Amount.Round(scale, mode)
	if err != nil {
		return Money{}, err
	}
	return Money{Amount: amount, Currency: m.Currency}, nil
}
