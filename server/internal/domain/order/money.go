package order

import (
	"math/big"
	"strings"
)

var ten = big.NewInt(10)

// Money represents a non-negative exact decimal currency amount with up to 34 significant digits.
type Money struct {
	coeff *big.Int
	scale int
}

var ZeroMoney = Money{coeff: big.NewInt(0), scale: 0}

// MoneyFromCoefficient constructs Money from a coefficient and a power-of-10 exponent:
// value = coeff * 10^exp10.
// Rejects negative values and enforces the 34 significant digits limit (Decimal128).
func MoneyFromCoefficient(coeff *big.Int, exp10 int) (Money, error) {
	if coeff == nil {
		coeff = big.NewInt(0)
	}
	if coeff.Sign() < 0 {
		return Money{}, ErrInvalidMoney
	}
	if coeff.Sign() == 0 {
		return Money{coeff: big.NewInt(0), scale: 0}, nil
	}

	if exp10 >= 0 {
		factor := new(big.Int).Exp(ten, big.NewInt(int64(exp10)), nil)
		c := new(big.Int).Mul(coeff, factor)
		if len(c.String()) > 34 {
			return Money{}, ErrMoneyOverflow
		}
		return Money{coeff: c, scale: 0}, nil
	}

	// exp10 < 0
	scale := -exp10
	c := new(big.Int).Set(coeff)
	div := new(big.Int)
	mod := new(big.Int)
	for scale > 0 {
		div.DivMod(c, ten, mod)
		if mod.Sign() == 0 {
			c.Set(div)
			scale--
		} else {
			break
		}
	}
	if len(c.String()) > 34 {
		return Money{}, ErrMoneyOverflow
	}
	return Money{coeff: c, scale: scale}, nil
}

// Int64 returns the integer value and true if Money is an integer that fits in int64.
// If Money has a fractional part or overflows int64, returns (0, false).
func (m Money) Int64() (int64, bool) {
	if m.coeff == nil {
		return 0, true
	}
	if m.scale > 0 {
		return 0, false
	}
	if !m.coeff.IsInt64() {
		return 0, false
	}
	return m.coeff.Int64(), true
}

// ParseMoney parses a decimal string into Money.
// Rejects negative amounts, NaN/Inf, empty strings, and malformed characters.
// Enforces a maximum of 34 significant digits (Decimal128 limit).
func ParseMoney(s string) (Money, error) {
	if len(s) == 0 {
		return Money{}, ErrInvalidMoney
	}

	// Validate character set and structure: only digits and at most one decimal point.
	dotIndex := -1
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if ch == '.' {
			if dotIndex != -1 {
				return Money{}, ErrInvalidMoney // Multiple dots
			}
			dotIndex = i
		} else if ch < '0' || ch > '9' {
			return Money{}, ErrInvalidMoney // Non-digit character
		}
	}

	var intPart, fracPart string
	if dotIndex == -1 {
		intPart = s
		fracPart = ""
	} else {
		// Must have at least one digit before and after the dot
		if dotIndex == 0 || dotIndex == len(s)-1 {
			return Money{}, ErrInvalidMoney
		}
		intPart = s[:dotIndex]
		fracPart = s[dotIndex+1:]
	}

	intTrimmed := strings.TrimLeft(intPart, "0")
	if intTrimmed == "" {
		intTrimmed = "0"
	}
	fracTrimmed := strings.TrimRight(fracPart, "0")

	// If value is 0
	if intTrimmed == "0" && fracTrimmed == "" {
		return Money{coeff: big.NewInt(0), scale: 0}, nil
	}

	if intTrimmed != "0" {
		fullStr := intTrimmed + fracTrimmed
		if len(fullStr) > 34 {
			return Money{}, ErrMoneyOverflow
		}
		coeff, ok := new(big.Int).SetString(fullStr, 10)
		if !ok {
			return Money{}, ErrInvalidMoney
		}
		return Money{coeff: coeff, scale: len(fracTrimmed)}, nil
	}

	// intTrimmed == "0" and fracTrimmed != ""
	coeffStr := strings.TrimLeft(fracTrimmed, "0")
	if len(coeffStr) > 34 {
		return Money{}, ErrMoneyOverflow
	}
	coeff, ok := new(big.Int).SetString(coeffStr, 10)
	if !ok {
		return Money{}, ErrInvalidMoney
	}
	return Money{coeff: coeff, scale: len(fracTrimmed)}, nil
}

// String formats the Money value as a canonical decimal string:
// no trailing zeroes, no decimal point if integer, no scientific notation.
func (m Money) String() string {
	if m.coeff == nil || m.coeff.Sign() == 0 {
		return "0"
	}
	s := m.coeff.String()
	if m.scale == 0 {
		return s
	}
	if len(s) <= m.scale {
		return "0." + strings.Repeat("0", m.scale-len(s)) + s
	}
	splitPos := len(s) - m.scale
	return s[:splitPos] + "." + s[splitPos:]
}

// MarshalJSON serializes Money as a JSON number without quotes.
func (m Money) MarshalJSON() ([]byte, error) {
	return []byte(m.String()), nil
}

// UnmarshalJSON deserializes Money from a JSON number or string.
func (m *Money) UnmarshalJSON(data []byte) error {
	str := strings.TrimSpace(string(data))
	if len(str) >= 2 && str[0] == '"' && str[len(str)-1] == '"' {
		str = str[1 : len(str)-1]
	}
	parsed, err := ParseMoney(str)
	if err != nil {
		return err
	}
	*m = parsed
	return nil
}

// IsZero returns true if the value is zero.
func (m Money) IsZero() bool {
	return m.coeff == nil || m.coeff.Sign() == 0
}

// Add adds another Money value and returns the exact sum.
// Returns ErrMoneyOverflow if the result exceeds 34 significant digits.
func (m Money) Add(other Money) (Money, error) {
	c1 := m.coeff
	if c1 == nil {
		c1 = big.NewInt(0)
	}
	c2 := other.coeff
	if c2 == nil {
		c2 = big.NewInt(0)
	}

	scale1 := m.scale
	scale2 := other.scale
	targetScale := scale1
	if scale2 > targetScale {
		targetScale = scale2
	}

	val1 := new(big.Int).Set(c1)
	if scale1 < targetScale {
		factor := new(big.Int).Exp(ten, big.NewInt(int64(targetScale-scale1)), nil)
		val1.Mul(val1, factor)
	}

	val2 := new(big.Int).Set(c2)
	if scale2 < targetScale {
		factor := new(big.Int).Exp(ten, big.NewInt(int64(targetScale-scale2)), nil)
		val2.Mul(val2, factor)
	}

	sum := new(big.Int).Add(val1, val2)

	// Canonicalize: trim trailing zeros while scale > 0
	div := new(big.Int)
	mod := new(big.Int)
	for targetScale > 0 {
		if sum.Sign() == 0 {
			targetScale = 0
			break
		}
		div.DivMod(sum, ten, mod)
		if mod.Sign() == 0 {
			sum.Set(div)
			targetScale--
		} else {
			break
		}
	}

	if sum.Sign() != 0 && len(sum.String()) > 34 {
		return Money{}, ErrMoneyOverflow
	}

	return Money{coeff: sum, scale: targetScale}, nil
}

// MulQty multiplies Money by an integer quantity.
// Returns ErrMoneyOverflow if the result exceeds 34 significant digits.
func (m Money) MulQty(qty int64) (Money, error) {
	if qty < 0 {
		return Money{}, ErrInvalidMoney
	}
	if qty == 0 || m.coeff == nil || m.coeff.Sign() == 0 {
		return Money{coeff: big.NewInt(0), scale: 0}, nil
	}

	prod := new(big.Int).Mul(m.coeff, big.NewInt(qty))
	scale := m.scale

	div := new(big.Int)
	mod := new(big.Int)
	for scale > 0 {
		div.DivMod(prod, ten, mod)
		if mod.Sign() == 0 {
			prod.Set(div)
			scale--
		} else {
			break
		}
	}

	if prod.Sign() != 0 && len(prod.String()) > 34 {
		return Money{}, ErrMoneyOverflow
	}

	return Money{coeff: prod, scale: scale}, nil
}

// Cmp compares m and other.
// Returns -1 if m < other, 0 if m == other, 1 if m > other.
func (m Money) Cmp(other Money) int {
	c1 := m.coeff
	if c1 == nil {
		c1 = big.NewInt(0)
	}
	c2 := other.coeff
	if c2 == nil {
		c2 = big.NewInt(0)
	}

	scale1 := m.scale
	scale2 := other.scale
	targetScale := scale1
	if scale2 > targetScale {
		targetScale = scale2
	}

	val1 := new(big.Int).Set(c1)
	if scale1 < targetScale {
		factor := new(big.Int).Exp(ten, big.NewInt(int64(targetScale-scale1)), nil)
		val1.Mul(val1, factor)
	}

	val2 := new(big.Int).Set(c2)
	if scale2 < targetScale {
		factor := new(big.Int).Exp(ten, big.NewInt(int64(targetScale-scale2)), nil)
		val2.Mul(val2, factor)
	}

	return val1.Cmp(val2)
}

// Equal returns true if m and other represent the exact same value.
func (m Money) Equal(other Money) bool {
	return m.Cmp(other) == 0
}
