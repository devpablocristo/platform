package money

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
)

var (
	ErrInvalidDecimal  = errors.New("money: invalid decimal")
	ErrInvalidScale    = errors.New("money: invalid scale")
	ErrUnsupportedScan = errors.New("money: unsupported scan value")
)

// RoundMode defines how Decimal.Round resolves discarded fractional digits.
type RoundMode int

const (
	RoundHalfUp RoundMode = iota
	RoundHalfEven
	RoundDown
)

// Decimal is an exact base-10 number represented as coeff * 10^-scale.
type Decimal struct {
	coeff big.Int
	scale int32
}

// Zero returns 0.
func Zero() Decimal {
	return Decimal{}
}

// ParseDecimal parses a finite decimal string. Exponents and floats are rejected.
func ParseDecimal(input string) (Decimal, error) {
	raw := strings.TrimSpace(input)
	if raw == "" {
		return Decimal{}, ErrInvalidDecimal
	}

	sign := 1
	switch raw[0] {
	case '-':
		sign = -1
		raw = raw[1:]
	case '+':
		raw = raw[1:]
	}
	if raw == "" {
		return Decimal{}, ErrInvalidDecimal
	}

	var digits strings.Builder
	var scale int32
	seenDot := false
	seenDigit := false

	for _, r := range raw {
		switch {
		case r >= '0' && r <= '9':
			seenDigit = true
			digits.WriteRune(r)
			if seenDot {
				scale++
			}
		case r == '.':
			if seenDot {
				return Decimal{}, ErrInvalidDecimal
			}
			seenDot = true
		default:
			return Decimal{}, ErrInvalidDecimal
		}
	}
	if !seenDigit {
		return Decimal{}, ErrInvalidDecimal
	}

	coeff := new(big.Int)
	if _, ok := coeff.SetString(digits.String(), 10); !ok {
		return Decimal{}, ErrInvalidDecimal
	}
	if sign < 0 {
		coeff.Neg(coeff)
	}
	return normalize(Decimal{coeff: *coeff, scale: scale}), nil
}

// MustParseDecimal parses a decimal and panics on invalid input.
func MustParseDecimal(input string) Decimal {
	value, err := ParseDecimal(input)
	if err != nil {
		panic(err)
	}
	return value
}

// NewDecimal creates a decimal from coeff * 10^-scale.
func NewDecimal(coeff int64, scale int32) (Decimal, error) {
	if scale < 0 {
		return Decimal{}, ErrInvalidScale
	}
	return normalize(Decimal{coeff: *big.NewInt(coeff), scale: scale}), nil
}

// Scale returns the decimal scale.
func (d Decimal) Scale() int32 {
	return d.scale
}

// Sign returns -1, 0, or 1.
func (d Decimal) Sign() int {
	return d.coeff.Sign()
}

// String returns the canonical decimal representation.
func (d Decimal) String() string {
	if d.coeff.Sign() == 0 {
		return "0"
	}

	abs := new(big.Int).Abs(&d.coeff).String()
	negative := d.coeff.Sign() < 0
	if d.scale == 0 {
		if negative {
			return "-" + abs
		}
		return abs
	}

	scale := int(d.scale)
	if len(abs) <= scale {
		abs = strings.Repeat("0", scale-len(abs)+1) + abs
	}
	point := len(abs) - scale
	out := abs[:point] + "." + abs[point:]
	if negative {
		return "-" + out
	}
	return out
}

// MarshalJSON encodes decimals as JSON strings to avoid binary float loss.
func (d Decimal) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

// UnmarshalJSON decodes a decimal from a JSON string.
func (d *Decimal) UnmarshalJSON(data []byte) error {
	var raw string
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	parsed, err := ParseDecimal(raw)
	if err != nil {
		return err
	}
	*d = parsed
	return nil
}

// Value returns a PostgreSQL NUMERIC-compatible string.
func (d Decimal) Value() (driver.Value, error) {
	return d.String(), nil
}

// Scan reads a PostgreSQL NUMERIC-compatible value.
func (d *Decimal) Scan(value any) error {
	switch v := value.(type) {
	case string:
		return d.scanString(v)
	case []byte:
		return d.scanString(string(v))
	case nil:
		*d = Zero()
		return nil
	default:
		return fmt.Errorf("%w: %T", ErrUnsupportedScan, value)
	}
}

func (d *Decimal) scanString(value string) error {
	parsed, err := ParseDecimal(value)
	if err != nil {
		return err
	}
	*d = parsed
	return nil
}

// Cmp compares two decimals.
func (d Decimal) Cmp(other Decimal) int {
	left, right := align(d, other)
	return left.Cmp(right)
}

// Add returns d + other.
func (d Decimal) Add(other Decimal) Decimal {
	left, right := align(d, other)
	left.Add(left, right)
	return normalize(Decimal{coeff: *left, scale: maxScale(d.scale, other.scale)})
}

// Sub returns d - other.
func (d Decimal) Sub(other Decimal) Decimal {
	left, right := align(d, other)
	left.Sub(left, right)
	return normalize(Decimal{coeff: *left, scale: maxScale(d.scale, other.scale)})
}

// Mul returns d * other.
func (d Decimal) Mul(other Decimal) Decimal {
	coeff := new(big.Int).Mul(&d.coeff, &other.coeff)
	return normalize(Decimal{coeff: *coeff, scale: d.scale + other.scale})
}

// Round returns d rounded to scale with mode.
func (d Decimal) Round(scale int32, mode RoundMode) (Decimal, error) {
	if scale < 0 {
		return Decimal{}, ErrInvalidScale
	}
	if scale >= d.scale {
		coeff := new(big.Int).Set(&d.coeff)
		coeff.Mul(coeff, pow10(int(scale-d.scale)))
		return Decimal{coeff: *coeff, scale: scale}, nil
	}

	diff := int(d.scale - scale)
	divisor := pow10(diff)
	quotient, remainder := new(big.Int), new(big.Int)
	quotient.QuoRem(&d.coeff, divisor, remainder)

	absRemainder := new(big.Int).Abs(remainder)
	if absRemainder.Sign() == 0 || mode == RoundDown {
		return normalize(Decimal{coeff: *quotient, scale: scale}), nil
	}

	twice := new(big.Int).Mul(absRemainder, big.NewInt(2))
	cmp := twice.Cmp(divisor)
	round := cmp > 0
	if cmp == 0 {
		switch mode {
		case RoundHalfUp:
			round = true
		case RoundHalfEven:
			round = new(big.Int).Abs(quotient).Bit(0) == 1
		default:
			return Decimal{}, fmt.Errorf("money: unsupported round mode %d", mode)
		}
	}
	if round {
		if d.coeff.Sign() < 0 {
			quotient.Sub(quotient, big.NewInt(1))
		} else {
			quotient.Add(quotient, big.NewInt(1))
		}
	}
	return normalize(Decimal{coeff: *quotient, scale: scale}), nil
}

func normalize(d Decimal) Decimal {
	if d.coeff.Sign() == 0 {
		return Decimal{}
	}
	ten := big.NewInt(10)
	zero := big.NewInt(0)
	for d.scale > 0 {
		q, r := new(big.Int), new(big.Int)
		q.QuoRem(&d.coeff, ten, r)
		if r.Cmp(zero) != 0 {
			break
		}
		d.coeff = *q
		d.scale--
	}
	return d
}

func align(left Decimal, right Decimal) (*big.Int, *big.Int) {
	scale := maxScale(left.scale, right.scale)
	l := new(big.Int).Set(&left.coeff)
	r := new(big.Int).Set(&right.coeff)
	if left.scale < scale {
		l.Mul(l, pow10(int(scale-left.scale)))
	}
	if right.scale < scale {
		r.Mul(r, pow10(int(scale-right.scale)))
	}
	return l, r
}

func maxScale(left int32, right int32) int32 {
	if left > right {
		return left
	}
	return right
}

func pow10(exp int) *big.Int {
	if exp <= 0 {
		return big.NewInt(1)
	}
	return new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(exp)), nil)
}
