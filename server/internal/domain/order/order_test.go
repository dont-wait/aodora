package order

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
)

type canonicalFixtureCase struct {
	Name          string        `json:"name"`
	Description   string        `json:"description"`
	Input         CheckoutInput `json:"input"`
	CanonicalJSON string        `json:"canonical_json"`
	RequestHash   string        `json:"request_hash"`
}

type canonicalFixture struct {
	Cases []canonicalFixtureCase `json:"cases"`
}

func TestNormalizeValidation(t *testing.T) {
	validAddress := ShippingAddressInput{
		Recipient: "Nguyễn An",
		Phone:     "0900000001",
		Line1:     "12 Nguyễn Trãi",
		City:      "TP. Hồ Chí Minh",
	}

	validItem := OrderItemInput{
		ProductID: "680000000000000000000001",
		SKU:       "LUA-DO-M",
		Quantity:  1,
	}

	// 1. Rejected payment methods
	for _, pm := range []string{"bank_transfer", "card", "e_wallet", "momo", "zalopay"} {
		input := CheckoutInput{
			PaymentMethod:   pm,
			ShippingAddress: validAddress,
			Items:           []OrderItemInput{validItem},
		}
		_, err := Normalize(input)
		var ifErr *InvalidFieldError
		if !errors.As(err, &ifErr) || ifErr.Field != "payment_method" {
			t.Errorf("expected InvalidFieldError on 'payment_method' for %s, got %v", pm, err)
		}
	}

	// 2. COD missing line1 or city
	codMissingLine1 := CheckoutInput{
		PaymentMethod: PaymentMethodCOD,
		ShippingAddress: ShippingAddressInput{
			Recipient: "Nguyễn An",
			Phone:     "0900000001",
			City:      "TP. Hồ Chí Minh",
		},
		Items: []OrderItemInput{validItem},
	}
	if _, err := Normalize(codMissingLine1); err == nil || !strings.Contains(err.Error(), "shipping_address.line1") {
		t.Errorf("expected error on shipping_address.line1, got %v", err)
	}

	codMissingCity := CheckoutInput{
		PaymentMethod: PaymentMethodCOD,
		ShippingAddress: ShippingAddressInput{
			Recipient: "Nguyễn An",
			Phone:     "0900000001",
			Line1:     "12 Nguyễn Trãi",
		},
		Items: []OrderItemInput{validItem},
	}
	if _, err := Normalize(codMissingCity); err == nil || !strings.Contains(err.Error(), "shipping_address.city") {
		t.Errorf("expected error on shipping_address.city, got %v", err)
	}

	// 3. Cash at counter only requires recipient and phone
	counterOnlyNamePhone := CheckoutInput{
		PaymentMethod: PaymentMethodCashAtCounter,
		ShippingAddress: ShippingAddressInput{
			Recipient: "Nguyễn An",
			Phone:     "0900000001",
		},
		Items: []OrderItemInput{validItem},
	}
	normCounter, err := Normalize(counterOnlyNamePhone)
	if err != nil {
		t.Fatalf("unexpected error for cash_at_counter minimal: %v", err)
	}
	if normCounter.ShippingAddress.Line1 != "" || normCounter.ShippingAddress.City != "" {
		t.Errorf("expected line1/city to be empty, got %+v", normCounter.ShippingAddress)
	}

	// Cash at counter keeps optional address fields if provided
	counterWithAddress := CheckoutInput{
		PaymentMethod: PaymentMethodCashAtCounter,
		ShippingAddress: ShippingAddressInput{
			Recipient: "Nguyễn An",
			Phone:     "0900000001",
			Line1:     "  12 Nguyễn Trãi  ",
			Ward:      "  Phường 1  ",
			City:      "  TP. HCM  ",
		},
		Items: []OrderItemInput{validItem},
	}
	normCounter2, err := Normalize(counterWithAddress)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if normCounter2.ShippingAddress.Line1 != "12 Nguyễn Trãi" || normCounter2.ShippingAddress.Ward != "Phường 1" {
		t.Errorf("optional fields were not trimmed/kept: %+v", normCounter2.ShippingAddress)
	}

	// 4. Whitespace-only strings treated as empty / rejected for required fields
	whitespaceAddress := CheckoutInput{
		PaymentMethod: PaymentMethodCOD,
		ShippingAddress: ShippingAddressInput{
			Recipient: "   ",
			Phone:     "0900000001",
			Line1:     "12 Nguyễn Trãi",
			City:      "TP. Hồ Chí Minh",
		},
		Items: []OrderItemInput{validItem},
	}
	if _, err := Normalize(whitespaceAddress); err == nil || !strings.Contains(err.Error(), "shipping_address.recipient") {
		t.Errorf("expected error on recipient for whitespace, got %v", err)
	}

	// 5. Quantity <= 0 or > 1,000,000
	qty0 := CheckoutInput{
		PaymentMethod:   PaymentMethodCashAtCounter,
		ShippingAddress: ShippingAddressInput{Recipient: "A", Phone: "1"},
		Items: []OrderItemInput{
			{ProductID: "680000000000000000000001", SKU: "S", Quantity: 0},
		},
	}
	if _, err := Normalize(qty0); err == nil || !strings.Contains(err.Error(), "quantity") {
		t.Errorf("expected error for quantity 0, got %v", err)
	}

	qtyNeg := CheckoutInput{
		PaymentMethod:   PaymentMethodCashAtCounter,
		ShippingAddress: ShippingAddressInput{Recipient: "A", Phone: "1"},
		Items: []OrderItemInput{
			{ProductID: "680000000000000000000001", SKU: "S", Quantity: -5},
		},
	}
	if _, err := Normalize(qtyNeg); err == nil || !strings.Contains(err.Error(), "quantity") {
		t.Errorf("expected error for quantity < 0, got %v", err)
	}

	qtyExceed := CheckoutInput{
		PaymentMethod:   PaymentMethodCashAtCounter,
		ShippingAddress: ShippingAddressInput{Recipient: "A", Phone: "1"},
		Items: []OrderItemInput{
			{ProductID: "680000000000000000000001", SKU: "S", Quantity: 1000001},
		},
	}
	if _, err := Normalize(qtyExceed); err == nil || !strings.Contains(err.Error(), "quantity") {
		t.Errorf("expected error for quantity > 1,000,000, got %v", err)
	}

	// 6. Merging duplicate items and overflow check
	dupItems := CheckoutInput{
		PaymentMethod:   PaymentMethodCashAtCounter,
		ShippingAddress: ShippingAddressInput{Recipient: "A", Phone: "1"},
		Items: []OrderItemInput{
			{ProductID: "680000000000000000000001", SKU: "SKU-A", Quantity: 500000},
			{ProductID: "680000000000000000000001", SKU: "SKU-A", Quantity: 500000},
		},
	}
	normDup, err := Normalize(dupItems)
	if err != nil {
		t.Fatalf("unexpected error merging duplicates: %v", err)
	}
	if len(normDup.Items) != 1 || normDup.Items[0].Quantity != 1000000 {
		t.Errorf("expected 1 item with quantity 1000000, got %+v", normDup.Items)
	}

	dupOverflow := CheckoutInput{
		PaymentMethod:   PaymentMethodCashAtCounter,
		ShippingAddress: ShippingAddressInput{Recipient: "A", Phone: "1"},
		Items: []OrderItemInput{
			{ProductID: "680000000000000000000001", SKU: "SKU-A", Quantity: 600000},
			{ProductID: "680000000000000000000001", SKU: "SKU-A", Quantity: 500000},
		},
	}
	if _, err := Normalize(dupOverflow); err == nil || !strings.Contains(err.Error(), "quantity") {
		t.Errorf("expected error for merged quantity exceeding limit, got %v", err)
	}

	// 7. Product ID normalization (trimmed, lowercased, validated 24 hex)
	upperProdID := CheckoutInput{
		PaymentMethod:   PaymentMethodCashAtCounter,
		ShippingAddress: ShippingAddressInput{Recipient: "A", Phone: "1"},
		Items: []OrderItemInput{
			{ProductID: "  68000000000000000000000A  ", SKU: "  SKU-A  ", Quantity: 1},
		},
	}
	normUpper, err := Normalize(upperProdID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if normUpper.Items[0].ProductID != "68000000000000000000000a" {
		t.Errorf("product_id was not lowercased: %s", normUpper.Items[0].ProductID)
	}
	if normUpper.Items[0].SKU != "SKU-A" {
		t.Errorf("SKU case was modified: %s", normUpper.Items[0].SKU)
	}
}

func TestCanonicalJSONAndFixture(t *testing.T) {
	fixtureData, err := os.ReadFile("../../../docs/fixtures/DW_CHECKOUT_CANONICAL.json")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	var fix canonicalFixture
	if err := json.Unmarshal(fixtureData, &fix); err != nil {
		t.Fatalf("failed to parse fixture: %v", err)
	}

	if len(fix.Cases) < 3 {
		t.Fatalf("fixture must have at least 3 cases, got %d", len(fix.Cases))
	}

	for _, tc := range fix.Cases {
		t.Run(tc.Name, func(t *testing.T) {
			norm, err := Normalize(tc.Input)
			if err != nil {
				t.Fatalf("Normalize failed: %v", err)
			}

			canonBytes, err := CanonicalJSON(norm)
			if err != nil {
				t.Fatalf("CanonicalJSON failed: %v", err)
			}

			if string(canonBytes) != tc.CanonicalJSON {
				t.Errorf("CanonicalJSON mismatch:\ngot:  %s\nwant: %s", string(canonBytes), tc.CanonicalJSON)
			}

			hash := RequestHash(canonBytes)
			if hash != tc.RequestHash {
				t.Errorf("RequestHash mismatch:\ngot:  %s\nwant: %s", hash, tc.RequestHash)
			}
		})
	}
}

func TestCanonicalEquivalenceAndDifference(t *testing.T) {
	// Two inputs with different item order and duplicate items that normalize to the same state
	inputA := CheckoutInput{
		PaymentMethod: PaymentMethodCashAtCounter,
		ShippingAddress: ShippingAddressInput{
			Recipient: "Nguyễn An",
			Phone:     "0900000001",
		},
		Items: []OrderItemInput{
			{ProductID: "680000000000000000000002", SKU: "SKU-B", Quantity: 1},
			{ProductID: "680000000000000000000001", SKU: "SKU-A", Quantity: 2},
			{ProductID: "680000000000000000000001", SKU: "SKU-A", Quantity: 1},
		},
		QuoteToken: "token-1",
	}

	inputB := CheckoutInput{
		PaymentMethod: PaymentMethodCashAtCounter,
		ShippingAddress: ShippingAddressInput{
			Recipient: "Nguyễn An",
			Phone:     "0900000001",
		},
		Items: []OrderItemInput{
			{ProductID: "680000000000000000000001", SKU: "SKU-A", Quantity: 3},
			{ProductID: "680000000000000000000002", SKU: "SKU-B", Quantity: 1},
		},
		QuoteToken: "token-1",
	}

	normA, err := Normalize(inputA)
	if err != nil {
		t.Fatal(err)
	}
	normB, err := Normalize(inputB)
	if err != nil {
		t.Fatal(err)
	}

	canonA, _ := CanonicalJSON(normA)
	canonB, _ := CanonicalJSON(normB)

	if string(canonA) != string(canonB) {
		t.Errorf("expected identical canonical JSON, got:\nA: %s\nB: %s", string(canonA), string(canonB))
	}
	hashA := RequestHash(canonA)
	hashB := RequestHash(canonB)
	if hashA != hashB {
		t.Errorf("expected identical hash, got %s != %s", hashA, hashB)
	}

	// Different quote_token produces different hash
	inputC := inputB
	inputC.QuoteToken = "token-different"
	normC, _ := Normalize(inputC)
	canonC, _ := CanonicalJSON(normC)
	hashC := RequestHash(canonC)
	if hashA == hashC {
		t.Errorf("expected different hash for different quote_token, but both were %s", hashA)
	}
}
