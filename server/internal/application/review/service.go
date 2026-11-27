package review

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"unicode/utf8"

	apporder "github.com/your-org/aodora/server/internal/application/order"
	domainorder "github.com/your-org/aodora/server/internal/domain/order"
)

var (
	ErrNotEligible = errors.New("customer is not eligible to review this product")
	ErrNotFound    = errors.New("review not found")
	ErrProduct     = errors.New("review product not found")
	ErrUnavailable = errors.New("review service unavailable")
	// ErrOrderAccess hides whether a guest order exists when the token does not match.
	ErrOrderAccess = errors.New("order not accessible with this credential")
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

// PublicReview never carries contact data, customer/order IDs or tokens.
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
	ProductID string `json:"product_id"`
	OrderID   string `json:"order_id"`
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

// GuestInput omits order_id: the guest route takes it from the URL path.
type GuestInput struct {
	Rating  int64   `json:"rating"`
	Content string  `json:"content"`
	Images  []Image `json:"images"`
}

const (
	SourceAccount = "account"
	SourceGuest   = "guest"
)

// AdminReview adds moderation context. It still omits email/phone and tokens.
type AdminReview struct {
	PublicReview
	ProductID   string `json:"product_id"`
	ProductName string `json:"product_name"`
	ProductSlug string `json:"product_slug"`
	Source      string `json:"source"`
	OrderNumber string `json:"order_number,omitempty"`
	UpdatedAt   string `json:"updated_at"`
}

type AdminFilter struct {
	ProductID string
	Rating    int64
	Page      int64
	Size      int64
}

type AdminPage struct {
	Items []AdminReview `json:"data"`
	Total int64         `json:"total"`
	Page  int64         `json:"page"`
	Size  int64         `json:"page_size"`
}

// Repository writes (Upsert, GuestUpsert, Delete) must recompute
// products.rating/review_count in the same transaction as the review change.
type Repository interface {
	List(context.Context, string, int64, int64) (Page, error)
	Own(context.Context, string, string) (*OwnReview, error)
	Upsert(context.Context, string, string, Input) (OwnReview, error)
	GuestReviews(context.Context, string) ([]OwnReview, error)
	GuestUpsert(context.Context, string, string, string, GuestInput) (OwnReview, error)
	AdminList(context.Context, AdminFilter) (AdminPage, error)
	Delete(context.Context, string) error
}

type PurchaseVerifier interface {
	VerifiedPurchase(context.Context, apporder.Viewer, string, string) (bool, error)
	Get(context.Context, apporder.Viewer, string) (*domainorder.Order, error)
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
	page, size = clampPage(page, size, 6, 50)
	return s.repo.List(ctx, productID, page, size)
}

func (s *Service) Own(ctx context.Context, customerID, productID string) (*OwnReview, error) {
	return s.repo.Own(ctx, customerID, productID)
}

// Submit trusts only the session customer ID; order ownership, delivery and
// product membership are checked against the stored order.
func (s *Service) Submit(ctx context.Context, customerID, productID string, input Input) (OwnReview, error) {
	if strings.TrimSpace(input.OrderID) == "" {
		return OwnReview{}, Invalid{Field: "order_id", Message: "Đơn hàng là bắt buộc"}
	}
	rating, content, images, err := normalize(input.Rating, input.Content, input.Images)
	if err != nil {
		return OwnReview{}, err
	}
	input.Rating, input.Content, input.Images = rating, content, images
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

// GuestReviews returns the reviews attached to one guest order. The guest
// must hold that order's access token; another order's token gives nothing.
func (s *Service) GuestReviews(ctx context.Context, guestHash, orderID string) ([]OwnReview, error) {
	if _, err := s.guestOrder(ctx, guestHash, orderID); err != nil {
		return nil, err
	}
	return s.repo.GuestReviews(ctx, orderID)
}

// SubmitGuest keeps one review per guest order and product; resubmitting
// updates it. Display name comes from the order, not from the request.
func (s *Service) SubmitGuest(ctx context.Context, guestHash, orderID, productID string, input GuestInput) (OwnReview, error) {
	rating, content, images, err := normalize(input.Rating, input.Content, input.Images)
	if err != nil {
		return OwnReview{}, err
	}
	input.Rating, input.Content, input.Images = rating, content, images
	order, err := s.guestOrder(ctx, guestHash, orderID)
	if err != nil {
		return OwnReview{}, err
	}
	verified, err := s.orders.VerifiedPurchase(ctx, apporder.Viewer{GuestHash: guestHash}, orderID, productID)
	if err != nil {
		return OwnReview{}, err
	}
	if !verified {
		return OwnReview{}, ErrNotEligible
	}
	return s.repo.GuestUpsert(ctx, orderID, productID, strings.TrimSpace(order.ShippingAddress.Recipient), input)
}

func (s *Service) guestOrder(ctx context.Context, guestHash, orderID string) (*domainorder.Order, error) {
	if s.orders == nil {
		return nil, ErrUnavailable
	}
	if guestHash == "" || strings.TrimSpace(orderID) == "" {
		return nil, ErrOrderAccess
	}
	order, err := s.orders.Get(ctx, apporder.Viewer{GuestHash: guestHash}, orderID)
	if errors.Is(err, apporder.ErrNotFound) || (err == nil && order == nil) {
		return nil, ErrOrderAccess
	}
	if err != nil {
		return nil, err
	}
	return order, nil
}

func (s *Service) AdminList(ctx context.Context, filter AdminFilter) (AdminPage, error) {
	if filter.Rating != 0 && (filter.Rating < 1 || filter.Rating > 5) {
		return AdminPage{}, Invalid{Field: "rating", Message: "Điểm lọc phải từ 1 đến 5"}
	}
	filter.Page, filter.Size = clampPage(filter.Page, filter.Size, 20, 100)
	return s.repo.AdminList(ctx, filter)
}

// AdminDelete is the moderation action agreed for Q-06: the schema has no
// hidden/approved state, so an inappropriate review is removed permanently.
func (s *Service) AdminDelete(ctx context.Context, id string) error {
	if strings.TrimSpace(id) == "" {
		return ErrNotFound
	}
	return s.repo.Delete(ctx, id)
}

func clampPage(page, size, fallback, max int64) (int64, int64) {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = fallback
	}
	if size > max {
		size = max
	}
	return page, size
}

func normalize(rating int64, content string, images []Image) (int64, string, []Image, error) {
	if rating < 1 || rating > 5 {
		return 0, "", nil, Invalid{Field: "rating", Message: "Điểm đánh giá phải từ 1 đến 5"}
	}
	content = strings.TrimSpace(content)
	if utf8.RuneCountInString(content) > 3000 {
		return 0, "", nil, Invalid{Field: "content", Message: "Nội dung tối đa 3000 ký tự"}
	}
	if len(images) > 5 {
		return 0, "", nil, Invalid{Field: "images", Message: "Tối đa 5 ảnh"}
	}
	// Never store null: the reviews validator requires an array.
	out := make([]Image, 0, len(images))
	for _, image := range images {
		image.Src, image.Alt = strings.TrimSpace(image.Src), strings.TrimSpace(image.Alt)
		if image.Src == "" || image.Alt == "" {
			return 0, "", nil, Invalid{Field: "images", Message: "Ảnh phải có URL và mô tả"}
		}
		u, err := url.Parse(image.Src)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil {
			return 0, "", nil, Invalid{Field: "images", Message: "Ảnh chỉ nhận URL http(s) hợp lệ"}
		}
		if utf8.RuneCountInString(image.Src) > 2048 || utf8.RuneCountInString(image.Alt) > 200 {
			return 0, "", nil, Invalid{Field: "images", Message: "URL hoặc mô tả ảnh quá dài"}
		}
		out = append(out, image)
	}
	return rating, content, out, nil
}
