package order

import (
	"encoding/json"
	"errors"
	"math/big"
	"strings"
	"testing"
)

func TestMoneyParseAndFormat(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantString string
	}{
		{"standard integer", "500000", "500000"},
		{"trailing zeroes removed", "500000.00", "500000"},
		{"trailing zero with fraction", "199999.50", "199999.5"},
		{"zero integer", "0", "0"},
		{"zero decimal", "0.000", "0"},
		{"leading zero integer", "00123", "123"},
		{"leading zero fraction", "0.00123", "0.00123"},
		{"exact decimal", "0.1", "0.1"},
		{"exact decimal 2", "0.2", "0.2"},
		{"large integer exceeding int64", "100000000000000000000", "100000000000000000000"},
		{"34 significant digits max", "1234567890123456789012345678901234", "1234567890123456789012345678901234"},
		{"34 significant digits with fraction", "1.234567890123456789012345678901234", "1.234567890123456789012345678901234"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m, err := ParseMoney(tc.input)
			if err != nil {
				t.Fatalf("unexpected error parsing %q: %v", tc.input, err)
			}
			if got := m.String(); got != tc.wantString {
				t.Errorf("m.String() = %q, want %q", got, tc.wantString)
			}
		})
	}
}

func TestMoneyAddition(t *testing.T) {
	// 0.1 + 0.2 = 0.3
	m1, err := ParseMoney("0.1")
	if err != nil {
		t.Fatal(err)
	}
	m2, err := ParseMoney("0.2")
	if err != nil {
		t.Fatal(err)
	}
	sum, err := m1.Add(m2)
	if err != nil {
		t.Fatalf("unexpected error adding: %v", err)
	}
	if sum.String() != "0.3" {
		t.Errorf("0.1 + 0.2 = %s, want 0.3", sum.String())
	}

	// 0.15 + 0.85 = 1
	m3, _ := ParseMoney("0.15")
	m4, _ := ParseMoney("0.85")
	sum2, err := m3.Add(m4)
	if err != nil {
		t.Fatal(err)
	}
	if sum2.String() != "1" {
		t.Errorf("0.15 + 0.85 = %s, want 1", sum2.String())
	}
}

func TestMoneyMultiplicationByQuantity(t *testing.T) {
	// 199999.5 * 3 = 599998.5
	m, err := ParseMoney("199999.5")
	if err != nil {
		t.Fatal(err)
	}
	prod, err := m.MulQty(3)
	if err != nil {
		t.Fatalf("unexpected error multiplying: %v", err)
	}
	if prod.String() != "599998.5" {
		t.Errorf("199999.5 * 3 = %s, want 599998.5", prod.String())
	}

	// multiply by 0 = 0
	prodZero, err := m.MulQty(0)
	if err != nil {
		t.Fatal(err)
	}
	if prodZero.String() != "0" || !prodZero.IsZero() {
		t.Errorf("m * 0 = %s, want 0", prodZero.String())
	}

	// negative quantity error
	if _, err := m.MulQty(-1); err == nil {
		t.Error("expected error for negative quantity, got nil")
	}
}

func TestMoneyOverflow(t *testing.T) {
	// 35 digits: overflow
	s35 := "1" + strings.Repeat("0", 34)
	if _, err := ParseMoney(s35); !errors.Is(err, ErrMoneyOverflow) {
		t.Errorf("expected ErrMoneyOverflow for 35 digits, got %v", err)
	}

	// 34 digits add 34 digits causing 35 digits overflow
	s34 := strings.Repeat("9", 34)
	m1, err := ParseMoney(s34)
	if err != nil {
		t.Fatal(err)
	}
	one, _ := ParseMoney("1")
	if _, err := m1.Add(one); !errors.Is(err, ErrMoneyOverflow) {
		t.Errorf("expected ErrMoneyOverflow on Add, got %v", err)
	}

	// 34 digits mul 2 causing overflow
	if _, err := m1.MulQty(2); !errors.Is(err, ErrMoneyOverflow) {
		t.Errorf("expected ErrMoneyOverflow on MulQty, got %v", err)
	}
}

func TestMoneyRejections(t *testing.T) {
	invalidInputs := []string{
		"",
		"-1",
		"-0.01",
		"+100",
		"NaN",
		"Inf",
		"+Inf",
		"-Inf",
		"abc",
		"12.3.4",
		"12a3",
		"1e5",
		"100 ",
		" 100",
		"100. ",
		".5",
		"5.",
	}

	for _, input := range invalidInputs {
		t.Run("invalid_"+input, func(t *testing.T) {
			_, err := ParseMoney(input)
			if err == nil {
				t.Errorf("expected error parsing %q, got nil", input)
			}
		})
	}
}

func TestMoneyJSON(t *testing.T) {
	type wrapper struct {
		Amount Money `json:"amount"`
	}

	m, err := ParseMoney("500000.00")
	if err != nil {
		t.Fatal(err)
	}
	w := wrapper{Amount: m}
	data, err := json.Marshal(w)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{"amount":500000}` {
		t.Errorf("json.Marshal = %s, want %s", string(data), `{"amount":500000}`)
	}

	m2, _ := ParseMoney("199999.5")
	w2 := wrapper{Amount: m2}
	data2, _ := json.Marshal(w2)
	if string(data2) != `{"amount":199999.5}` {
		t.Errorf("json.Marshal = %s, want %s", string(data2), `{"amount":199999.5}`)
	}

	// Unmarshal
	var w3 wrapper
	if err := json.Unmarshal([]byte(`{"amount":500000}`), &w3); err != nil {
		t.Fatal(err)
	}
	if w3.Amount.String() != "500000" {
		t.Errorf("unmarshaled = %s, want 500000", w3.Amount.String())
	}
}

func TestMoneyCmpAndEqual(t *testing.T) {
	m1, _ := ParseMoney("100")
	m2, _ := ParseMoney("100.00")
	m3, _ := ParseMoney("200")
	m4, _ := ParseMoney("50")

	if !m1.Equal(m2) {
		t.Errorf("%s should equal %s", m1, m2)
	}
	if m1.Cmp(m3) >= 0 {
		t.Errorf("%s should be less than %s", m1, m3)
	}
	if m1.Cmp(m4) <= 0 {
		t.Errorf("%s should be greater than %s", m1, m4)
	}
}

func TestMoneyFromCoefficient(t *testing.T) {
	// 500 * 10^3 = 500000
	m1, err := MoneyFromCoefficient(big.NewInt(500), 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m1.String() != "500000" {
		t.Errorf("500 * 10^3 = %s, want 500000", m1.String())
	}

	// 1999995 * 10^-1 = 199999.5
	m2, err := MoneyFromCoefficient(big.NewInt(1999995), -1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m2.String() != "199999.5" {
		t.Errorf("1999995 * 10^-1 = %s, want 199999.5", m2.String())
	}

	// 12300 * 10^-2 = 123 (trailing zeroes reduced)
	m3, err := MoneyFromCoefficient(big.NewInt(12300), -2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m3.String() != "123" {
		t.Errorf("12300 * 10^-2 = %s, want 123", m3.String())
	}

	// 0 * 10^0 = 0
	m0, err := MoneyFromCoefficient(big.NewInt(0), 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m0.String() != "0" {
		t.Errorf("0 * 10^0 = %s, want 0", m0.String())
	}

	// nil coeff = 0
	mnil, err := MoneyFromCoefficient(nil, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mnil.String() != "0" {
		t.Errorf("nil * 10^5 = %s, want 0", mnil.String())
	}

	// Negative coeff rejected
	if _, err := MoneyFromCoefficient(big.NewInt(-10), 2); !errors.Is(err, ErrInvalidMoney) {
		t.Errorf("expected ErrInvalidMoney for negative coeff, got %v", err)
	}

	// Overflow: > 34 digits
	bigCoeff, _ := new(big.Int).SetString("1"+strings.Repeat("0", 34), 10)
	if _, err := MoneyFromCoefficient(bigCoeff, 0); !errors.Is(err, ErrMoneyOverflow) {
		t.Errorf("expected ErrMoneyOverflow, got %v", err)
	}
}

func TestMoneyInt64(t *testing.T) {
	mInt, _ := ParseMoney("500000")
	val, ok := mInt.Int64()
	if !ok || val != 500000 {
		t.Errorf("expected (500000, true), got (%d, %v)", val, ok)
	}

	mZero := ZeroMoney
	val, ok = mZero.Int64()
	if !ok || val != 0 {
		t.Errorf("expected (0, true), got (%d, %v)", val, ok)
	}

	mDec, _ := ParseMoney("199999.5")
	val, ok = mDec.Int64()
	if ok || val != 0 {
		t.Errorf("expected (0, false) for decimal, got (%d, %v)", val, ok)
	}

	// Max int64
	mMaxInt, _ := ParseMoney("9223372036854775807")
	val, ok = mMaxInt.Int64()
	if !ok || val != 9223372036854775807 {
		t.Errorf("expected max int64, got (%d, %v)", val, ok)
	}

	// Max int64 + 1
	mOverflowInt, _ := ParseMoney("9223372036854775808")
	val, ok = mOverflowInt.Int64()
	if ok || val != 0 {
		t.Errorf("expected (0, false) for overflow int64, got (%d, %v)", val, ok)
	}
}
