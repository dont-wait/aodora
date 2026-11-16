package order

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"unicode/utf8"
)

const (
	PaymentMethodCOD           = "cod"
	PaymentMethodCashAtCounter = "cash_at_counter"

	MaxItemQuantity  = 1000000
	MaxDistinctItems = 100
)

type ShippingAddressInput struct {
	Recipient string `json:"recipient"`
	Phone     string `json:"phone"`
	Line1     string `json:"line1,omitempty"`
	Ward      string `json:"ward,omitempty"`
	District  string `json:"district,omitempty"`
	City      string `json:"city,omitempty"`
}

type OrderItemInput struct {
	ProductID string `json:"product_id"`
	SKU       string `json:"sku"`
	Quantity  int64  `json:"quantity"`
}

type CheckoutInput struct {
	PaymentMethod   string               `json:"payment_method"`
	ShippingAddress ShippingAddressInput `json:"shipping_address"`
	Items           []OrderItemInput     `json:"items"`
	Note            string               `json:"note,omitempty"`
	QuoteToken      string               `json:"quote_token,omitempty"`
}

type NormalizedAddress struct {
	Recipient string `json:"recipient"`
	Phone     string `json:"phone"`
	Line1     string `json:"line1,omitempty"`
	Ward      string `json:"ward,omitempty"`
	District  string `json:"district,omitempty"`
	City      string `json:"city,omitempty"`
}

type NormalizedItem struct {
	ProductID   string `json:"product_id"`
	SKU         string `json:"sku"`
	Quantity    int64  `json:"quantity"`
	SourceIndex int    `json:"-"`
}

type NormalizedCheckout struct {
	PaymentMethod   string            `json:"payment_method"`
	ShippingAddress NormalizedAddress `json:"shipping_address"`
	Items           []NormalizedItem  `json:"items"`
	Note            string            `json:"note,omitempty"`
	QuoteToken      string            `json:"quote_token,omitempty"`
}

func isValidHex24(s string) bool {
	if len(s) != 24 {
		return false
	}
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if (ch < '0' || ch > '9') && (ch < 'a' || ch > 'f') {
			return false
		}
	}
	return true
}

// Normalize validates and normalizes checkout inputs:
// - trims all strings
// - product_id is trimmed, lowercased, and verified as 24-character hex
// - SKU is trimmed while preserving case
// - optional strings that become empty after trimming are treated as absent
// - payment_method must be "cod" or "cash_at_counter"
// - recipient and phone are required for all methods
// - for COD, line1 and city are also required
// - items must not be empty
// - quantities must be integers >= 1 and sum per (product_id, sku) <= 1,000,000
// - merges duplicate (product_id, sku) lines with int64 overflow check
// - limits distinct items after merge to <= 100
// - sorts items by product_id ascending, then sku ascending
func Normalize(input CheckoutInput) (*NormalizedCheckout, error) {
	pm := strings.TrimSpace(input.PaymentMethod)
	if pm != PaymentMethodCOD && pm != PaymentMethodCashAtCounter {
		return nil, NewInvalidFieldError("payment_method", "Chỉ hỗ trợ thanh toán khi giao hàng hoặc tại quầy")
	}

	recipient := strings.TrimSpace(input.ShippingAddress.Recipient)
	if recipient == "" {
		return nil, NewInvalidFieldError("shipping_address.recipient", "Tên người nhận không được để trống")
	}
	if utf8.RuneCountInString(recipient) > 120 {
		return nil, NewInvalidFieldError("shipping_address.recipient", "Tên người nhận không được vượt quá 120 ký tự")
	}

	phone := strings.TrimSpace(input.ShippingAddress.Phone)
	if phone == "" {
		return nil, NewInvalidFieldError("shipping_address.phone", "Số điện thoại không được để trống")
	}
	if utf8.RuneCountInString(phone) > 30 {
		return nil, NewInvalidFieldError("shipping_address.phone", "Số điện thoại không được vượt quá 30 ký tự")
	}

	line1 := strings.TrimSpace(input.ShippingAddress.Line1)
	if utf8.RuneCountInString(line1) > 200 {
		return nil, NewInvalidFieldError("shipping_address.line1", "Địa chỉ nhận hàng không được vượt quá 200 ký tự")
	}

	ward := strings.TrimSpace(input.ShippingAddress.Ward)
	if utf8.RuneCountInString(ward) > 200 {
		return nil, NewInvalidFieldError("shipping_address.ward", "Phường/xã không được vượt quá 200 ký tự")
	}

	district := strings.TrimSpace(input.ShippingAddress.District)
	if utf8.RuneCountInString(district) > 200 {
		return nil, NewInvalidFieldError("shipping_address.district", "Quận/huyện không được vượt quá 200 ký tự")
	}

	city := strings.TrimSpace(input.ShippingAddress.City)
	if utf8.RuneCountInString(city) > 200 {
		return nil, NewInvalidFieldError("shipping_address.city", "Tỉnh/thành phố không được vượt quá 200 ký tự")
	}

	if pm == PaymentMethodCOD {
		if line1 == "" {
			return nil, NewInvalidFieldError("shipping_address.line1", "Địa chỉ nhận hàng không được để trống")
		}
		if city == "" {
			return nil, NewInvalidFieldError("shipping_address.city", "Tỉnh/thành phố không được để trống")
		}
	}

	if len(input.Items) == 0 {
		return nil, NewInvalidFieldError("items", "Danh sách sản phẩm không được để trống")
	}

	type itemKey struct {
		productID string
		sku       string
	}
	mergedIndex := make(map[itemKey]int)
	var mergedItems []NormalizedItem

	for i, it := range input.Items {
		prodID := strings.ToLower(strings.TrimSpace(it.ProductID))
		if !isValidHex24(prodID) {
			return nil, NewInvalidFieldError(fmt.Sprintf("items[%d].product_id", i), "Mã sản phẩm không hợp lệ (phải là 24 ký tự hex)")
		}

		sku := strings.TrimSpace(it.SKU)
		if sku == "" {
			return nil, NewInvalidFieldError(fmt.Sprintf("items[%d].sku", i), "Mã SKU không được để trống")
		}
		if utf8.RuneCountInString(sku) > 100 {
			return nil, NewInvalidFieldError(fmt.Sprintf("items[%d].sku", i), "Mã SKU không được vượt quá 100 ký tự")
		}

		if it.Quantity < 1 {
			return nil, NewInvalidFieldError(fmt.Sprintf("items[%d].quantity", i), "Số lượng sản phẩm phải lớn hơn hoặc bằng 1")
		}
		if it.Quantity > MaxItemQuantity {
			return nil, NewInvalidFieldError(fmt.Sprintf("items[%d].quantity", i), "Số lượng sản phẩm vượt quá giới hạn 1.000.000")
		}

		k := itemKey{productID: prodID, sku: sku}
		if idx, exists := mergedIndex[k]; exists {
			existingQty := mergedItems[idx].Quantity
			if math.MaxInt64-existingQty < it.Quantity {
				return nil, NewInvalidFieldError(fmt.Sprintf("items[%d].quantity", i), "Số lượng sản phẩm bị tràn số")
			}
			newTotal := existingQty + it.Quantity
			if newTotal > MaxItemQuantity {
				return nil, NewInvalidFieldError(fmt.Sprintf("items[%d].quantity", i), "Tổng số lượng cho mỗi sản phẩm vượt quá giới hạn 1.000.000")
			}
			mergedItems[idx].Quantity = newTotal
		} else {
			mergedIndex[k] = len(mergedItems)
			mergedItems = append(mergedItems, NormalizedItem{
				ProductID:   prodID,
				SKU:         sku,
				Quantity:    it.Quantity,
				SourceIndex: i,
			})
		}
	}

	if len(mergedItems) > MaxDistinctItems {
		return nil, NewInvalidFieldError("items", "Đơn hàng không được vượt quá 100 sản phẩm khác nhau")
	}

	sort.Slice(mergedItems, func(i, j int) bool {
		if mergedItems[i].ProductID != mergedItems[j].ProductID {
			return mergedItems[i].ProductID < mergedItems[j].ProductID
		}
		return mergedItems[i].SKU < mergedItems[j].SKU
	})

	note := strings.TrimSpace(input.Note)
	if utf8.RuneCountInString(note) > 1000 {
		return nil, NewInvalidFieldError("note", "Ghi chú không được vượt quá 1000 ký tự")
	}

	return &NormalizedCheckout{
		PaymentMethod: pm,
		ShippingAddress: NormalizedAddress{
			Recipient: recipient,
			Phone:     phone,
			Line1:     line1,
			Ward:      ward,
			District:  district,
			City:      city,
		},
		Items:      mergedItems,
		Note:       note,
		QuoteToken: strings.TrimSpace(input.QuoteToken),
	}, nil
}

type canonicalRequest struct {
	PaymentMethod   string                   `json:"payment_method"`
	ShippingAddress canonicalShippingAddress `json:"shipping_address"`
	Items           []canonicalItem          `json:"items"`
	Note            string                   `json:"note,omitempty"`
	QuoteToken      string                   `json:"quote_token,omitempty"`
}

type canonicalShippingAddress struct {
	Recipient string `json:"recipient"`
	Phone     string `json:"phone"`
	Line1     string `json:"line1,omitempty"`
	Ward      string `json:"ward,omitempty"`
	District  string `json:"district,omitempty"`
	City      string `json:"city,omitempty"`
}

type canonicalItem struct {
	ProductID string `json:"product_id"`
	SKU       string `json:"sku"`
	Quantity  int64  `json:"quantity"`
}

// CanonicalJSON encodes the normalized checkout into canonical JSON format:
// UTF-8, no extra whitespace, fixed field order, omits empty optional fields,
// does NOT escape HTML characters (<>&).
func CanonicalJSON(norm *NormalizedCheckout) ([]byte, error) {
	if norm == nil {
		return nil, errors.New("normalized checkout cannot be nil")
	}
	req := canonicalRequest{
		PaymentMethod: norm.PaymentMethod,
		ShippingAddress: canonicalShippingAddress{
			Recipient: norm.ShippingAddress.Recipient,
			Phone:     norm.ShippingAddress.Phone,
			Line1:     norm.ShippingAddress.Line1,
			Ward:      norm.ShippingAddress.Ward,
			District:  norm.ShippingAddress.District,
			City:      norm.ShippingAddress.City,
		},
		Items:      make([]canonicalItem, len(norm.Items)),
		Note:       norm.Note,
		QuoteToken: norm.QuoteToken,
	}
	for i, it := range norm.Items {
		req.Items[i] = canonicalItem{
			ProductID: it.ProductID,
			SKU:       it.SKU,
			Quantity:  it.Quantity,
		}
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(req); err != nil {
		return nil, err
	}
	b := buf.Bytes()
	if len(b) > 0 && b[len(b)-1] == '\n' {
		b = b[:len(b)-1]
	}
	return b, nil
}

// RequestHash computes the SHA-256 lowercase hex string of the canonical JSON bytes.
func RequestHash(canonicalJSON []byte) string {
	h := sha256.Sum256(canonicalJSON)
	return hex.EncodeToString(h[:])
}
