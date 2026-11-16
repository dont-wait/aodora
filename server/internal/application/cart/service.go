package cart

import (
	"context"
	"errors"
	"strings"
)

var (
	ErrInvalidSelection = errors.New("invalid cart selection")
	ErrItemUnavailable  = errors.New("cart item unavailable")
	ErrConflict         = errors.New("cart changed")
)

const (
	IssueProductUnavailable = "PRODUCT_UNAVAILABLE"
	IssueSKUNotFound        = "SKU_NOT_FOUND"
	IssueOutOfStock         = "OUT_OF_STOCK"
	IssueQuantityExceeded   = "QUANTITY_EXCEEDS_STOCK"
	IssuePriceChanged       = "PRICE_CHANGED"
)

type Invalid struct {
	Field   string
	Message string
}

func (e Invalid) Error() string { return e.Field + ": " + e.Message }

type Selection struct {
	ProductID       string `json:"product_id"`
	ProductSlug     string `json:"product_slug"`
	SKU             string `json:"sku"`
	Color           string `json:"color"`
	Size            string `json:"size"`
	Quantity        int64  `json:"quantity"`
	FulfillmentMode string `json:"fulfillment_mode"`
}

type Item struct {
	Selection
	Name        string `json:"name,omitempty"`
	Image       string `json:"image,omitempty"`
	UnitPrice   int64  `json:"unit_price"`
	LineTotal   int64  `json:"line_total"`
	Stock       int64  `json:"stock"`
	Available   bool   `json:"available"`
	Issue       string `json:"issue,omitempty"`
	IssueDetail string `json:"issue_detail,omitempty"`
}

type Cart struct {
	Items     []Item `json:"items"`
	ItemCount int64  `json:"item_count"`
	Subtotal  int64  `json:"subtotal"`
	Issues    []Item `json:"issues,omitempty"`
}

type Repository interface {
	Cart(context.Context, string) (Cart, error)
	AddItem(context.Context, string, Selection) (Cart, error)
	UpdateItem(context.Context, string, Selection) (Cart, error)
	RemoveItem(context.Context, string, string, string) (Cart, error)
	Merge(context.Context, string, string, []Selection) (Cart, error)
	Validate(context.Context, []Selection) (Cart, error)
}

type Service struct{ repo Repository }

func NewService(repo Repository) *Service { return &Service{repo: repo} }

func (s *Service) Cart(ctx context.Context, userID string) (Cart, error) {
	return s.repo.Cart(ctx, userID)
}

func (s *Service) AddItem(ctx context.Context, userID string, selection Selection) (Cart, error) {
	if err := validateSelection(selection); err != nil {
		return Cart{}, err
	}
	return s.repo.AddItem(ctx, userID, selection)
}

func (s *Service) UpdateItem(ctx context.Context, userID string, selection Selection) (Cart, error) {
	if err := validateSelection(selection); err != nil {
		return Cart{}, err
	}
	return s.repo.UpdateItem(ctx, userID, selection)
}

func (s *Service) RemoveItem(ctx context.Context, userID, productID, sku string) (Cart, error) {
	if strings.TrimSpace(productID) == "" || strings.TrimSpace(sku) == "" {
		return Cart{}, Invalid{Field: "item", Message: "Sản phẩm và SKU là bắt buộc"}
	}
	return s.repo.RemoveItem(ctx, userID, productID, sku)
}

func (s *Service) Merge(ctx context.Context, userID, requestID string, selections []Selection) (Cart, error) {
	if len(selections) > 100 {
		return Cart{}, Invalid{Field: "items", Message: "Giỏ hàng có quá nhiều dòng"}
	}
	if strings.TrimSpace(requestID) == "" || len(requestID) > 100 {
		return Cart{}, Invalid{Field: "request_id", Message: "Mã hợp nhất không hợp lệ"}
	}
	for _, selection := range selections {
		if err := validateSelection(selection); err != nil {
			return Cart{}, err
		}
	}
	return s.repo.Merge(ctx, userID, requestID, selections)
}

func (s *Service) Validate(ctx context.Context, selections []Selection) (Cart, error) {
	if len(selections) > 100 {
		return Cart{}, Invalid{Field: "items", Message: "Giỏ hàng có quá nhiều dòng"}
	}
	for _, selection := range selections {
		if err := validateSelection(selection); err != nil {
			return Cart{}, err
		}
	}
	return s.repo.Validate(ctx, selections)
}

func validateSelection(selection Selection) error {
	if strings.TrimSpace(selection.ProductID) == "" || strings.TrimSpace(selection.SKU) == "" {
		return Invalid{Field: "item", Message: "Sản phẩm và SKU là bắt buộc"}
	}
	if !strings.EqualFold(selection.FulfillmentMode, "ready") {
		return Invalid{Field: "fulfillment_mode", Message: "Giỏ hàng chỉ nhận sản phẩm may sẵn"}
	}
	if selection.Quantity < 1 || selection.Quantity > 1000000 {
		return Invalid{Field: "quantity", Message: "Số lượng phải từ 1 đến 1000000"}
	}
	return nil
}
