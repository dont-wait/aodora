package order

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

type QuoteItem struct {
	ProductID string `json:"product_id"`
	SKU       string `json:"sku"`
	Quantity  int64  `json:"quantity"`
	UnitPrice string `json:"unit_price"`
}

type QuotePayload struct {
	V             int         `json:"v"`
	Exp           int64       `json:"exp"`
	PaymentMethod string      `json:"payment_method"`
	Items         []QuoteItem `json:"items"`
	ShippingFee   string      `json:"shipping_fee"`
	Subtotal      string      `json:"subtotal"`
	Total         string      `json:"total"`
}

type QuoteSigner struct {
	secret []byte
}

func NewQuoteSigner(secret []byte) (*QuoteSigner, error) {
	if len(secret) < 32 {
		return nil, errors.New("quote secret must be at least 32 bytes")
	}
	sec := make([]byte, len(secret))
	copy(sec, secret)
	return &QuoteSigner{secret: sec}, nil
}

// Sign serializes and cryptographically signs the QuotePayload using HMAC-SHA256.
func (s *QuoteSigner) Sign(payload QuotePayload) (string, error) {
	payload.V = 1
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	mac := hmac.New(sha256.New, s.secret)
	mac.Write(payloadBytes)
	sig := mac.Sum(nil)

	token := base64.RawURLEncoding.EncodeToString(payloadBytes) + "." + base64.RawURLEncoding.EncodeToString(sig)
	return token, nil
}

// Verify decodes and validates a quote token against the current time.
// Returns ErrQuoteInvalid for corrupted or tampered tokens, and ErrQuoteExpired if expired.
func (s *QuoteSigner) Verify(token string, now time.Time) (*QuotePayload, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return nil, ErrQuoteInvalid
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, ErrQuoteInvalid
	}

	sigBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, ErrQuoteInvalid
	}

	mac := hmac.New(sha256.New, s.secret)
	mac.Write(payloadBytes)
	expectedSig := mac.Sum(nil)

	if !hmac.Equal(sigBytes, expectedSig) {
		return nil, ErrQuoteInvalid
	}

	var payload QuotePayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return nil, ErrQuoteInvalid
	}

	if payload.V != 1 {
		return nil, ErrQuoteInvalid
	}

	if now.Unix() >= payload.Exp {
		return nil, ErrQuoteExpired
	}

	return &payload, nil
}

// CompareQuoteChoices compares the quote's payment_method and items against the current normalized order selection.
// If any choice differs (payment method, item count, product_id, sku, quantity), it returns ErrQuoteChanged.
func CompareQuoteChoices(quote *QuotePayload, paymentMethod string, items []NormalizedItem) error {
	if quote == nil {
		return ErrQuoteInvalid
	}
	if quote.PaymentMethod != paymentMethod {
		return ErrQuoteChanged
	}
	if len(quote.Items) != len(items) {
		return ErrQuoteChanged
	}
	for i := range quote.Items {
		if quote.Items[i].ProductID != items[i].ProductID ||
			quote.Items[i].SKU != items[i].SKU ||
			quote.Items[i].Quantity != items[i].Quantity {
			return ErrQuoteChanged
		}
	}
	return nil
}
