package review

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"unicode/utf8"

	apporder "github.com/your-org/aodora/server/internal/application/order"
)

var (
	ErrNotEligible = errors.New("customer is not eligible to review this product")
	ErrNotFound    = errors.New("review not found")
	ErrProduct     = errors.New("review product not found")
	ErrUnavailable = errors.New("review service unavailable")
)

type Invalid struct {
	Field   string
	Message string
}

func (e Invalid) Error() string { return e.Field + ": " + e.Message }

type Image struct {
	Src string `json:"src" bson:"src"`
	Alt string `json:"alt" bson:"alt"`
}

type PublicReview struct {
	ID               string  `json:"id"`
	Rating           int64   `json:"rating"`
	Content          string  `json:"content,omitempty"`
	DisplayName      string  `json:"display_name"`
	Images           []Image `json:"images"`
	VerifiedPurchase bool    `json:"verified_purchase"`
	CreatedAt        string  `json:"created_at"`
}

type OwnReview struct {
	PublicReview
	OrderID string `json:"order_id"`
}

type Page struct {
	Items   []PublicReview `json:"items"`
	Total   int64          `json:"total"`
	Page    int64          `json:"page"`
	Size    int64          `json:"page_size"`
	Average float64        `json:"average"`
}

type Input struct {
	OrderID string  `json:"order_id"`
	Rating  int64   `json:"rating"`
	Content string  `json:"content"`
	Images  []Image `json:"images"`
}

type Repository interface {
	List(context.Context, string, int64, int64) (Page, error)
	Own(context.Context, string, string) (*OwnReview, error)
	Upsert(context.Context, string, string, Input) (OwnReview, error)
}

type PurchaseVerifier interface {
	VerifiedPurchase(context.Context, apporder.Viewer, string, string) (bool, error)
}

type Service struct {
	repo   Repository
	orders PurchaseVerifier
}

func NewService(repo Repository, orders PurchaseVerifier) *Service {
	return &Service{repo: repo, orders: orders}
}

func (s *Service) List(ctx context.Context, productID string, page, size int64) (Page, error) {
	if strings.TrimSpace(productID) == "" {
		return Page{}, Invalid{Field: "product_id", Message: "Sản phẩm là bắt buộc"}
	}
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 6
	}
	if size > 50 {
		size = 50
	}
	return s.repo.List(ctx, productID, page, size)
}

func (s *Service) Own(ctx context.Context, customerID, productID string) (*OwnReview, error) {
	return s.repo.Own(ctx, customerID, productID)
}

func (s *Service) Submit(ctx context.Context, customerID, productID string, input Input) (OwnReview, error) {
	input.Content = strings.TrimSpace(input.Content)
	for i := range input.Images {
		input.Images[i].Src = strings.TrimSpace(input.Images[i].Src)
		input.Images[i].Alt = strings.TrimSpace(input.Images[i].Alt)
	}
	if err := validateInput(input); err != nil {
		return OwnReview{}, err
	}
	if s.orders == nil {
		return OwnReview{}, ErrUnavailable
	}
	verified, err := s.orders.VerifiedPurchase(ctx, apporder.Viewer{CustomerID: customerID}, input.OrderID, productID)
	if err != nil {
		return OwnReview{}, err
	}
	if !verified {
		return OwnReview{}, ErrNotEligible
	}
	return s.repo.Upsert(ctx, customerID, productID, input)
}

func validateInput(input Input) error {
	if strings.TrimSpace(input.OrderID) == "" {
		return Invalid{Field: "order_id", Message: "Đơn hàng là bắt buộc"}
	}
	if input.Rating < 1 || input.Rating > 5 {
		return Invalid{Field: "rating", Message: "Điểm đánh giá phải từ 1 đến 5"}
	}
	input.Content = strings.TrimSpace(input.Content)
	if utf8.RuneCountInString(input.Content) > 3000 {
		return Invalid{Field: "content", Message: "Nội dung tối đa 3000 ký tự"}
	}
	if len(input.Images) > 5 {
		return Invalid{Field: "images", Message: "Tối đa 5 ảnh"}
	}
	for i, image := range input.Images {
		if strings.TrimSpace(image.Src) == "" || strings.TrimSpace(image.Alt) == "" {
			return Invalid{Field: "images", Message: "Ảnh phải có URL và mô tả"}
		}
		u, err := url.Parse(image.Src)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil {
			return Invalid{Field: "images", Message: "Ảnh chỉ nhận URL http(s) hợp lệ"}
		}
		if utf8.RuneCountInString(image.Alt) > 200 {
			return Invalid{Field: "images", Message: "Mô tả ảnh quá dài"}
		}
		_ = i
	}
	return nil
}
