package order

import (
	"crypto/rand"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestQuoteSignerConstructor(t *testing.T) {
	// Secret < 32 bytes rejected
	shortSecret := make([]byte, 31)
	if _, err := NewQuoteSigner(shortSecret); err == nil {
		t.Error("expected error for secret < 32 bytes, got nil")
	}

	// Secret >= 32 bytes accepted
	validSecret := make([]byte, 32)
	rand.Read(validSecret)
	signer, err := NewQuoteSigner(validSecret)
	if err != nil {
		t.Fatalf("unexpected error for valid secret: %v", err)
	}
	if signer == nil {
		t.Fatal("signer is nil")
	}
}

func TestQuoteSignAndVerify(t *testing.T) {
	secret := make([]byte, 32)
	for i := range secret {
		secret[i] = byte(i + 1)
	}
	signer, err := NewQuoteSigner(secret)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Unix(1700000000, 0)
	payload := QuotePayload{
		V:             1,
		Exp:           now.Add(15 * time.Minute).Unix(),
		PaymentMethod: "cod",
		Items: []QuoteItem{
			{ProductID: "680000000000000000000001", SKU: "SKU-A", Quantity: 2, UnitPrice: "250000"},
		},
		ShippingFee: "30000",
		Subtotal:    "500000",
		Total:       "530000",
	}

	token, err := signer.Sign(payload)
	if err != nil {
		t.Fatalf("Sign failed: %v", err)
	}

	// 1. Round-trip verify
	verified, err := signer.Verify(token, now)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if verified.PaymentMethod != payload.PaymentMethod || verified.Total != payload.Total {
		t.Errorf("verified payload mismatch: %+v", verified)
	}

	// 2. Tampered signature (corrupting 1 byte)
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		t.Fatalf("expected 2 parts, got %d", len(parts))
	}
	tamperedSig := parts[1]
	if tamperedSig[0] == 'A' {
		tamperedSig = "B" + tamperedSig[1:]
	} else {
		tamperedSig = "A" + tamperedSig[1:]
	}
	if _, err := signer.Verify(parts[0]+"."+tamperedSig, now); !errors.Is(err, ErrQuoteInvalid) {
		t.Errorf("expected ErrQuoteInvalid for tampered signature, got %v", err)
	}

	// 3. Tampered payload
	tamperedPayload := parts[0]
	if tamperedPayload[0] == 'A' {
		tamperedPayload = "B" + tamperedPayload[1:]
	} else {
		tamperedPayload = "A" + tamperedPayload[1:]
	}
	if _, err := signer.Verify(tamperedPayload+"."+parts[1], now); !errors.Is(err, ErrQuoteInvalid) {
		t.Errorf("expected ErrQuoteInvalid for tampered payload, got %v", err)
	}

	// 4. Expired token
	expiredTime := time.Unix(payload.Exp+1, 0)
	if _, err := signer.Verify(token, expiredTime); !errors.Is(err, ErrQuoteExpired) {
		t.Errorf("expected ErrQuoteExpired for expired quote, got %v", err)
	}

	// 5. Different secret
	otherSecret := make([]byte, 32)
	for i := range otherSecret {
		otherSecret[i] = byte(i + 2)
	}
	otherSigner, _ := NewQuoteSigner(otherSecret)
	if _, err := otherSigner.Verify(token, now); !errors.Is(err, ErrQuoteInvalid) {
		t.Errorf("expected ErrQuoteInvalid for different secret, got %v", err)
	}

	// 6. Malformed tokens (missing dot, garbage)
	if _, err := signer.Verify("garbage-no-dot", now); !errors.Is(err, ErrQuoteInvalid) {
		t.Errorf("expected ErrQuoteInvalid for token without dot, got %v", err)
	}
	if _, err := signer.Verify("a.b.c", now); !errors.Is(err, ErrQuoteInvalid) {
		t.Errorf("expected ErrQuoteInvalid for token with 3 parts, got %v", err)
	}
}

func TestCompareQuoteChoices(t *testing.T) {
	quote := &QuotePayload{
		PaymentMethod: "cod",
		Items: []QuoteItem{
			{ProductID: "680000000000000000000001", SKU: "SKU-A", Quantity: 2, UnitPrice: "250000"},
			{ProductID: "680000000000000000000002", SKU: "SKU-B", Quantity: 1, UnitPrice: "100000"},
		},
	}

	matchingItems := []NormalizedItem{
		{ProductID: "680000000000000000000001", SKU: "SKU-A", Quantity: 2},
		{ProductID: "680000000000000000000002", SKU: "SKU-B", Quantity: 1},
	}

	// Match
	if err := CompareQuoteChoices(quote, "cod", matchingItems); err != nil {
		t.Errorf("unexpected error for matching choices: %v", err)
	}

	// Different payment method
	if err := CompareQuoteChoices(quote, "cash_at_counter", matchingItems); !errors.Is(err, ErrQuoteChanged) {
		t.Errorf("expected ErrQuoteChanged for different payment method, got %v", err)
	}

	// Different item count
	if err := CompareQuoteChoices(quote, "cod", matchingItems[:1]); !errors.Is(err, ErrQuoteChanged) {
		t.Errorf("expected ErrQuoteChanged for different item count, got %v", err)
	}

	// Different quantity
	diffQtyItems := []NormalizedItem{
		{ProductID: "680000000000000000000001", SKU: "SKU-A", Quantity: 3},
		{ProductID: "680000000000000000000002", SKU: "SKU-B", Quantity: 1},
	}
	if err := CompareQuoteChoices(quote, "cod", diffQtyItems); !errors.Is(err, ErrQuoteChanged) {
		t.Errorf("expected ErrQuoteChanged for different quantity, got %v", err)
	}

	// Different SKU
	diffSKUItems := []NormalizedItem{
		{ProductID: "680000000000000000000001", SKU: "SKU-DIFF", Quantity: 2},
		{ProductID: "680000000000000000000002", SKU: "SKU-B", Quantity: 1},
	}
	if err := CompareQuoteChoices(quote, "cod", diffSKUItems); !errors.Is(err, ErrQuoteChanged) {
		t.Errorf("expected ErrQuoteChanged for different SKU, got %v", err)
	}

	// Different ProductID
	diffProdItems := []NormalizedItem{
		{ProductID: "680000000000000000000099", SKU: "SKU-A", Quantity: 2},
		{ProductID: "680000000000000000000002", SKU: "SKU-B", Quantity: 1},
	}
	if err := CompareQuoteChoices(quote, "cod", diffProdItems); !errors.Is(err, ErrQuoteChanged) {
		t.Errorf("expected ErrQuoteChanged for different product_id, got %v", err)
	}
}
