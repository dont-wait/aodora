// Package catalogadmin validates owner-managed catalog writes. It deliberately
// excludes order, review and customer data from all write contracts.
package catalogadmin

import (
	"context"
	"encoding/hex"
	"errors"
	"regexp"
	"strings"
)

var (
	ErrNotFound             = errors.New("catalog entity not found")
	ErrConflict             = errors.New("catalog write conflict")
	ErrTransactionsRequired = errors.New("category reparent requires MongoDB transactions")
)

type Invalid struct{ Field, Message string }

func (e Invalid) Error() string { return e.Field + ": " + e.Message }

type Image struct {
	Src string `json:"src" bson:"src"`
	Alt string `json:"alt" bson:"alt"`
}
type Color struct {
	Name string `json:"name" bson:"name"`
	Hex  string `json:"hex" bson:"hex"`
}
type Accessory struct {
	ID          string `json:"id" bson:"id"`
	Name        string `json:"name" bson:"name"`
	Description string `json:"description" bson:"description"`
	Price       int64  `json:"price" bson:"price"`
	Image       *Image `json:"image,omitempty" bson:"image,omitempty"`
	Tag         string `json:"tag,omitempty" bson:"tag,omitempty"`
	TagVariant  string `json:"tag_variant,omitempty" bson:"tag_variant,omitempty"`
}
type CategoryInput struct {
	Name            string `json:"name"`
	Slug            string `json:"slug"`
	Description     string `json:"description"`
	ParentID        string `json:"parent_id"`
	IsActive        bool   `json:"is_active"`
	SortOrder       int64  `json:"sort_order"`
	Image           *Image `json:"image"`
	ExpectedVersion *int64 `json:"expected_version,omitempty"`
}
type Variant struct {
	SKU   string `json:"sku" bson:"sku"`
	Color string `json:"color" bson:"color"`
	Size  string `json:"size" bson:"size"`
	Stock int64  `json:"stock" bson:"stock"`
}
type VariantDetails struct {
	Color string `json:"color"`
	Size  string `json:"size"`
}

// ProductInput is a full metadata replacement. Neither the aggregate stock,
// individual variant stock, ratings nor reviews can be modified with this DTO.
type ProductInput struct {
	Name            string      `json:"name"`
	Slug            string      `json:"slug"`
	CategoryID      string      `json:"category_id"`
	Description     string      `json:"description"`
	Material        string      `json:"material"`
	Price           int64       `json:"price"`
	CompareAtPrice  *int64      `json:"compare_at_price"`
	Images          []string    `json:"images"`
	Image           *Image      `json:"image"`
	Gallery         []Image     `json:"gallery"`
	Colors          []Color     `json:"colors"`
	Accessories     []Accessory `json:"accessories"`
	IsActive        bool        `json:"is_active"`
	ExpectedVersion *int64      `json:"expected_version,omitempty"`
}
type ProductCreate struct {
	ProductInput
	Variants []Variant `json:"variants"`
}
type InventoryDelta struct {
	Delta     int64  `json:"delta"`
	RequestID string `json:"request_id"`
}

type ProductPage struct {
	Items      []map[string]any `json:"items"`
	Total      int64            `json:"total"`
	Page       int64            `json:"page"`
	PageSize   int64            `json:"page_size"`
	TotalPages int64            `json:"total_pages"`
}

type Repository interface {
	Categories(context.Context) ([]map[string]any, error)
	Products(context.Context, int64, int64) (ProductPage, error)
	// ProductsBySlug and SKUOwners back the bulk import (lookups only, no writes).
	ProductsBySlug(context.Context, []string) ([]map[string]any, error)
	SKUOwners(context.Context, []string) (map[string]string, error)
	// ProductState is the current stored state used by validation (images, colours, SKUs).
	ProductState(context.Context, string) (ProductState, error)
	CategoryExists(context.Context, string) (bool, error)
	CreateCategory(context.Context, CategoryInput) (map[string]any, error)
	UpdateCategory(context.Context, string, CategoryInput) (map[string]any, error)
	CreateProduct(context.Context, ProductCreate) (map[string]any, error)
	UpdateProduct(context.Context, string, ProductInput) (map[string]any, error)
	AddVariant(context.Context, string, Variant) (map[string]any, error)
	UpdateVariant(context.Context, string, string, VariantDetails) (map[string]any, error)
	AdjustInventory(context.Context, string, string, int64, string) (map[string]any, error)
	ProductContent(context.Context, string) (ContentRecord, error)
	UpdateProductContent(context.Context, string, ProductContent) (ContentRecord, error)
}
type Service struct {
	repo        Repository
	imagePrefix string
}

// Option configures a Service.
type Option func(*Service)

// WithImagePrefix makes the service accept only images stored under the given public bucket URL
// (plus URLs already present in the record being edited).
func WithImagePrefix(prefix string) Option {
	return func(s *Service) { s.imagePrefix = strings.TrimRight(strings.TrimSpace(prefix), "/") }
}

func NewService(repo Repository, opts ...Option) *Service {
	s := &Service{repo: repo}
	for _, opt := range opts {
		opt(s)
	}
	return s
}
func (s *Service) Categories(ctx context.Context) ([]map[string]any, error) {
	return s.repo.Categories(ctx)
}
func (s *Service) Products(ctx context.Context, page, pageSize int64) (ProductPage, error) {
	if page < 1 || page > 10000 {
		return ProductPage{}, Invalid{Field: "page", Message: "Trang phải từ 1 đến 10000"}
	}
	if pageSize < 1 || pageSize > 100 {
		return ProductPage{}, Invalid{Field: "page_size", Message: "Kích thước trang phải từ 1 đến 100"}
	}
	return s.repo.Products(ctx, page, pageSize)
}

func (s *Service) checkCategory(ctx context.Context, id string) error {
	exists, err := s.repo.CategoryExists(ctx, id)
	if err != nil {
		return err
	}
	if !exists {
		return Invalid{"category_id", "Danh mục không tồn tại"}
	}
	return nil
}

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-8][0-9a-fA-F]{3}-[89aAbB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)

func expectedVersion(v *int64) error {
	if v == nil || *v < 0 {
		return Invalid{"expected_version", "Phải cung cấp phiên bản metadata không âm"}
	}
	return nil
}
func canonicalObjectID(v string) string {
	if len(v) != 24 {
		return ""
	}
	if _, err := hex.DecodeString(v); err != nil {
		return ""
	}
	return strings.ToLower(v)
}

// ProductState is the stored state of one product, read for validation only.
type ProductState struct {
	ID       string
	Slug     string
	Active   bool
	Images   []string // every image URL on the record (main, gallery, images[], accessories)
	Colors   []string
	Variants []StateVariant
}
type StateVariant struct {
	SKU, Color, Size string
	Stock            int64
}
