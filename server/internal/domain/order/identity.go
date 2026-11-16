package order

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"regexp"
	"strings"
)

var idempotencyKeyRegex = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

// ValidateGuestAccessToken validates a guest access token:
// - must be exactly 43 characters
// - base64url without padding (RFC 4648 §5)
// - strictly decodes to exactly 32 bytes
// - canonical encoding: re-encoding must produce the identical string (no non-zero padding bits)
// Returns the SHA-256 lowercase hex hash of the decoded 32 bytes, and the scope "guest:<hash>".
func ValidateGuestAccessToken(token string) (hashHex string, scope string, err error) {
	if len(token) != 43 {
		return "", "", ErrInvalidGuestToken
	}
	// Reject padding or standard base64 characters
	if strings.ContainsRune(token, '=') || strings.ContainsRune(token, '+') || strings.ContainsRune(token, '/') {
		return "", "", ErrInvalidGuestToken
	}

	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return "", "", ErrInvalidGuestToken
	}
	if len(raw) != 32 {
		return "", "", ErrInvalidGuestToken
	}

	// Canonicality check: re-encoding must match token
	reencoded := base64.RawURLEncoding.EncodeToString(raw)
	if reencoded != token {
		return "", "", ErrInvalidGuestToken
	}

	h := sha256.Sum256(raw)
	hashHex = hex.EncodeToString(h[:])
	scope = "guest:" + hashHex
	return hashHex, scope, nil
}

// ValidateCustomerScope validates a customer ObjectId (24 lowercase hex characters)
// and returns the scope "customer:<customerID>".
func ValidateCustomerScope(customerID string) (scope string, err error) {
	if len(customerID) != 24 {
		return "", ErrInvalidScope
	}
	for i := 0; i < len(customerID); i++ {
		ch := customerID[i]
		if (ch < '0' || ch > '9') && (ch < 'a' || ch > 'f') {
			return "", ErrInvalidScope
		}
	}
	return "customer:" + customerID, nil
}

// ValidateIdempotencyKey validates and canonicalizes an Idempotency-Key.
// Trims whitespace, converts to lowercase, and validates UUID format:
// `^[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`.
func ValidateIdempotencyKey(key string) (canonicalKey string, err error) {
	trimmed := strings.TrimSpace(key)
	lowered := strings.ToLower(trimmed)
	if !idempotencyKeyRegex.MatchString(lowered) {
		return "", NewInvalidFieldError("Idempotency-Key", "Idempotency-Key phải là UUID hợp lệ (phiên bản 1-8)")
	}
	return lowered, nil
}
