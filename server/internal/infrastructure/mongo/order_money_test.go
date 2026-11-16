package mongo

import (
	"testing"

	domainorder "github.com/your-org/aodora/server/internal/domain/order"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestDecodeMoneyRejectsDouble(t *testing.T) {
	cases := []any{
		float64(199999.5),
		float32(199999.5),
		float64(0),
		float64(-10),
	}
	for _, c := range cases {
		_, err := decodeMoney(c)
		if err == nil {
			t.Fatalf("expected error decoding float/double %T(%v), got nil", c, c)
		}
	}
}

func TestDecodeMoneyPositiveExponent(t *testing.T) {
	d, err := bson.ParseDecimal128("1E+3")
	if err != nil {
		t.Fatalf("parse decimal128: %v", err)
	}
	m, err := decodeMoney(d)
	if err != nil {
		t.Fatalf("decodeMoney(1E+3) failed: %v", err)
	}
	if m.String() != "1000" {
		t.Fatalf("expected 1000, got %s", m.String())
	}

	d2, err := bson.ParseDecimal128("2.5E+2")
	if err != nil {
		t.Fatalf("parse decimal128: %v", err)
	}
	m2, err := decodeMoney(d2)
	if err != nil {
		t.Fatalf("decodeMoney(2.5E+2) failed: %v", err)
	}
	if m2.String() != "250" {
		t.Fatalf("expected 250, got %s", m2.String())
	}
}

func TestDecodeMoneyRejectsNegativeNaNAndInf(t *testing.T) {
	// Negative int/int32/int64
	intCases := []any{int32(-1), int64(-100), int(-50)}
	for _, c := range intCases {
		if _, err := decodeMoney(c); err == nil {
			t.Fatalf("expected error for negative int %T(%v), got nil", c, c)
		}
	}

	// Negative Decimal128
	dNeg, _ := bson.ParseDecimal128("-199999.5")
	if _, err := decodeMoney(dNeg); err == nil {
		t.Fatalf("expected error for negative Decimal128, got nil")
	}

	// NaN
	dNaN, _ := bson.ParseDecimal128("NaN")
	if _, err := decodeMoney(dNaN); err == nil {
		t.Fatalf("expected error for NaN Decimal128, got nil")
	}

	// +Infinity
	dInf, _ := bson.ParseDecimal128("Infinity")
	if _, err := decodeMoney(dInf); err == nil {
		t.Fatalf("expected error for Infinity Decimal128, got nil")
	}

	// -Infinity
	dNegInf, _ := bson.ParseDecimal128("-Infinity")
	if _, err := decodeMoney(dNegInf); err == nil {
		t.Fatalf("expected error for -Infinity Decimal128, got nil")
	}
}

func TestRoundTripDecimalAndIntegerMoney(t *testing.T) {
	// 1. Decimal 199999.5 round-trip <=> Decimal128
	mDec, err := domainorder.ParseMoney("199999.5")
	if err != nil {
		t.Fatalf("parse money: %v", err)
	}
	encodedDec, err := encodeMoney(mDec)
	if err != nil {
		t.Fatalf("encodeMoney decimal failed: %v", err)
	}
	d128, ok := encodedDec.(bson.Decimal128)
	if !ok {
		t.Fatalf("expected bson.Decimal128, got %T", encodedDec)
	}
	if d128.String() != "199999.5" {
		t.Fatalf("expected Decimal128 string 199999.5, got %s", d128.String())
	}
	decodedDec, err := decodeMoney(encodedDec)
	if err != nil {
		t.Fatalf("decodeMoney decimal failed: %v", err)
	}
	if decodedDec.String() != "199999.5" {
		t.Fatalf("expected decoded decimal 199999.5, got %s", decodedDec.String())
	}

	// 2. Integer 200000 round-trip <=> int64
	mInt, err := domainorder.ParseMoney("200000")
	if err != nil {
		t.Fatalf("parse money: %v", err)
	}
	encodedInt, err := encodeMoney(mInt)
	if err != nil {
		t.Fatalf("encodeMoney integer failed: %v", err)
	}
	i64, ok := encodedInt.(int64)
	if !ok {
		t.Fatalf("expected int64, got %T", encodedInt)
	}
	if i64 != 200000 {
		t.Fatalf("expected int64 200000, got %d", i64)
	}
	decodedInt, err := decodeMoney(encodedInt)
	if err != nil {
		t.Fatalf("decodeMoney integer failed: %v", err)
	}
	if decodedInt.String() != "200000" {
		t.Fatalf("expected decoded int 200000, got %s", decodedInt.String())
	}
}
