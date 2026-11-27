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
	OrderID string `json:"order_id"`
	// Hidden is true when the shop owner hid the review from the public list.
	Hidden bool `json:"hidden"`
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

// AdminReview adds moderation context. It still omits email/phone and tokens.
type AdminReview struct {
	PublicReview
	ProductID   string `json:"product_id"`
	ProductName string `json:"product_name"`
	ProductSlug string `json:"product_slug"`
	OrderNumber string `json:"order_number,omitempty"`
	UpdatedAt   string `json:"updated_at"`
	Hidden      bool   `json:"hidden"`
}

// ProductReviewStatus tells the order list where to send a customer to
// review a product. Slug is empty when the product is no longer public.
type ProductReviewStatus struct {
	ProductID string `json:"product_id"`
	Slug      string `json:"slug"`
	Reviewed  bool   `json:"reviewed"`
}

// MaxStatusProducts bounds one status lookup (a page of orders).
const MaxStatusProducts = 200

type AdminFilter struct {
	ProductID string
	Rating    int64
	// Hidden nil lists every review; otherwise only hidden or only visible ones.
	Hidden *bool
	Page   int64
	Size   int64
}

type AdminPage struct {
	Items []AdminReview `json:"data"`
	Total int64         `json:"total"`
	Page  int64         `json:"page"`
	Size  int64         `json:"page_size"`
}

// Repository writes (Upsert, SetHidden, Delete) must recompute
// products.rating/review_count in the same transaction as the review change.
type Repository interface {
	List(context.Context, string, int64, int64) (Page, error)
	Own(context.Context, string, string) (*OwnReview, error)
	Upsert(context.Context, string, string, Input) (OwnReview, error)
	Statuses(context.Context, string, []string) ([]ProductReviewStatus, error)
	AdminList(context.Context, AdminFilter) (AdminPage, error)
	SetHidden(context.Context, string, bool) error
	Delete(context.Context, string) error
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
	page, size = clampPage(page, size, 6, 50)
	return s.repo.List(ctx, productID, page, size)
}

func (s *Service) Own(ctx context.Context, customerID, productID string) (*OwnReview, error) {
	return s.repo.Own(ctx, customerID, productID)
}

// Submit is the only review write path: reviewing requires a signed-in
// account (Q-03); guests can only read public reviews. Only the session
// customer ID is trusted; order ownership, delivery and product membership
// are checked against the stored order.
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

// Statuses reports, for the session customer, whether each product is
// already reviewed and its public slug. Duplicate IDs are collapsed.
func (s *Service) Statuses(ctx context.Context, customerID string, productIDs []string) ([]ProductReviewStatus, error) {
	seen := map[string]bool{}
	unique := make([]string, 0, len(productIDs))
	for _, id := range productIDs {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		unique = append(unique, id)
	}
	if len(unique) == 0 {
		return []ProductReviewStatus{}, nil
	}
	if len(unique) > MaxStatusProducts {
		return nil, Invalid{Field: "product_id", Message: "Quá nhiều sản phẩm trong một lần kiểm tra"}
	}
	return s.repo.Statuses(ctx, customerID, unique)
}

func (s *Service) AdminList(ctx context.Context, filter AdminFilter) (AdminPage, error) {
	if filter.Rating != 0 && (filter.Rating < 1 || filter.Rating > 5) {
		return AdminPage{}, Invalid{Field: "rating", Message: "Điểm lọc phải từ 1 đến 5"}
	}
	filter.Page, filter.Size = clampPage(filter.Page, filter.Size, 20, 100)
	return s.repo.AdminList(ctx, filter)
}

// AdminSetHidden hides or shows a review (Q-06). Hidden reviews stay in the
// database and are excluded from the public list and product aggregates.
func (s *Service) AdminSetHidden(ctx context.Context, id string, hidden bool) error {
	if strings.TrimSpace(id) == "" {
		return ErrNotFound
	}
	return s.repo.SetHidden(ctx, id, hidden)
}

// AdminDelete removes a review permanently (Q-06); prefer AdminSetHidden
// when the review may need to be restored.
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
