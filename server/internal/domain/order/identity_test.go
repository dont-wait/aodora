package order

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

type accessFixture struct {
	TestOnly        bool   `json:"test_only"`
	DecodedHex      string `json:"decoded_hex"`
	GuestAccessHash string `json:"guest_access_hash"`
	CheckoutScope   string `json:"checkout_scope"`
	TokenEncoding   string `json:"token_encoding"`
}

func TestGuestAccessTokenFixture(t *testing.T) {
	// Read fixture
	data, err := os.ReadFile("../../../docs/fixtures/DW_CHECKOUT_ACCESS.json")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}
	var fix accessFixture
	if err := json.Unmarshal(data, &fix); err != nil {
		t.Fatalf("failed to unmarshal fixture: %v", err)
	}

	// Generate 0..31 bytes at runtime (do NOT commit static token string)
	raw := make([]byte, 32)
	for i := 0; i < 32; i++ {
		raw[i] = byte(i)
	}

	token := base64.RawURLEncoding.EncodeToString(raw)

	hashHex, scope, err := ValidateGuestAccessToken(token)
	if err != nil {
		t.Fatalf("ValidateGuestAccessToken failed: %v", err)
	}

	if hashHex != fix.GuestAccessHash {
		t.Errorf("hash = %s, want %s", hashHex, fix.GuestAccessHash)
	}
	if scope != fix.CheckoutScope {
		t.Errorf("scope = %s, want %s", scope, fix.CheckoutScope)
	}
}

func TestGuestAccessTokenRejections(t *testing.T) {
	// Valid 32-byte raw
	raw := make([]byte, 32)
	validToken := base64.RawURLEncoding.EncodeToString(raw)

	// 1. Padding '=' rejected
	paddedToken := validToken + "="
	if _, _, err := ValidateGuestAccessToken(paddedToken); err == nil {
		t.Error("expected error for token with padding '='")
	}

	// 2. 42 characters rejected
	short42 := validToken[:42]
	if _, _, err := ValidateGuestAccessToken(short42); err == nil {
		t.Error("expected error for 42-char token")
	}

	// 3. 44 characters rejected
	long44 := validToken + "A"
	if _, _, err := ValidateGuestAccessToken(long44); err == nil {
		t.Error("expected error for 44-char token")
	}

	// 4. Characters '+' and '/' rejected
	tokenWithPlus := validToken[:42] + "+"
	if _, _, err := ValidateGuestAccessToken(tokenWithPlus); err == nil {
		t.Error("expected error for token with '+'")
	}
	tokenWithSlash := validToken[:42] + "/"
	if _, _, err := ValidateGuestAccessToken(tokenWithSlash); err == nil {
		t.Error("expected error for token with '/'")
	}

	// 5. 31 bytes or 33 bytes rejected
	raw31 := make([]byte, 31)
	token31 := base64.RawURLEncoding.EncodeToString(raw31)
	if _, _, err := ValidateGuestAccessToken(token31); err == nil {
		t.Error("expected error for 31-byte token")
	}
	raw33 := make([]byte, 33)
	token33 := base64.RawURLEncoding.EncodeToString(raw33)
	if _, _, err := ValidateGuestAccessToken(token33); err == nil {
		t.Error("expected error for 33-byte token")
	}

	// 6. Non-canonical 43-character token (unused bits at end are non-zero)
	// In 32-byte base64url, byte 31 is encoded in chars 42 and 43.
	// 43 chars * 6 bits = 258 bits. 32 bytes = 256 bits.
	// The 43rd char has 4 data bits and 2 unused bits.
	// If the raw bytes are all 0:
	// The canonical token ends with 'A' (bits 000000).
	// If we replace the 43rd char with 'B' (bits 000001, unused bit is 1):
	nonCanonicalToken := strings.Repeat("A", 42) + "B"
	if _, _, err := ValidateGuestAccessToken(nonCanonicalToken); err == nil {
		t.Error("expected error for non-canonical token with non-zero trailing bits")
	}
}

func TestCustomerScopeValidation(t *testing.T) {
	// Valid 24 hex lowercase
	validID := "680000000000000000000001"
	scope, err := ValidateCustomerScope(validID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if scope != "customer:680000000000000000000001" {
		t.Errorf("scope = %s, want customer:680000000000000000000001", scope)
	}

	// Uppercase rejected
	upperID := "68000000000000000000000A"
	if _, err := ValidateCustomerScope(upperID); err == nil {
		t.Error("expected error for uppercase hex customer ID")
	}

	// Length != 24 rejected
	shortID := "68000000000000000000001"
	if _, err := ValidateCustomerScope(shortID); err == nil {
		t.Error("expected error for 23-char customer ID")
	}

	// Non-hex characters rejected
	invalidCharID := "68000000000000000000000z"
	if _, err := ValidateCustomerScope(invalidCharID); err == nil {
		t.Error("expected error for non-hex character in customer ID")
	}
}

func TestIdempotencyKeyValidation(t *testing.T) {
	// Valid UUID v4
	validV4 := "ad577a65-cd42-46eb-ae8a-98b5f1186571"
	canon, err := ValidateIdempotencyKey(validV4)
	if err != nil {
		t.Fatalf("unexpected error for valid v4: %v", err)
	}
	if canon != validV4 {
		t.Errorf("canon = %s, want %s", canon, validV4)
	}

	// Uppercase converted to lowercase
	upper := "AD577A65-CD42-46EB-AE8A-98B5F1186571"
	canonUpper, err := ValidateIdempotencyKey(upper)
	if err != nil {
		t.Fatalf("unexpected error for uppercase UUID: %v", err)
	}
	if canonUpper != validV4 {
		t.Errorf("canonUpper = %s, want %s", canonUpper, validV4)
	}

	// Leading/trailing whitespace trimmed
	withSpace := "  ad577a65-cd42-46eb-ae8a-98b5f1186571 \n"
	canonSpace, err := ValidateIdempotencyKey(withSpace)
	if err != nil {
		t.Fatalf("unexpected error for padded UUID: %v", err)
	}
	if canonSpace != validV4 {
		t.Errorf("canonSpace = %s, want %s", canonSpace, validV4)
	}

	// Version 0 rejected
	v0 := "ad577a65-cd42-06eb-ae8a-98b5f1186571"
	if _, err := ValidateIdempotencyKey(v0); err == nil {
		t.Error("expected error for version 0 UUID")
	}

	// Version 9 rejected
	v9 := "ad577a65-cd42-96eb-ae8a-98b5f1186571"
	if _, err := ValidateIdempotencyKey(v9); err == nil {
		t.Error("expected error for version 9 UUID")
	}

	// Invalid variant rejected (RFC 4122 variant 1 requires 8, 9, a, or b. 'c' is variant 2)
	badVariant := "ad577a65-cd42-46eb-ce8a-98b5f1186571"
	if _, err := ValidateIdempotencyKey(badVariant); err == nil {
		t.Error("expected error for invalid variant UUID")
	}

	// Arbitrary non-UUID string rejected
	if _, err := ValidateIdempotencyKey("not-a-uuid"); err == nil {
		t.Error("expected error for non-uuid string")
	}
}
