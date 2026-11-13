// Package catalogadmin validates owner-managed catalog writes. It deliberately
// excludes order, review and customer data from all write contracts.
package catalogadmin

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"
)

var (
	ErrNotFound = errors.New("catalog entity not found")
	ErrConflict = errors.New("catalog write conflict")
)

type Invalid struct{ Field, Message string }

func (e Invalid) Error() string { return e.Field + ": " + e.Message }

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

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
	Image       Image  `json:"image" bson:"image"`
}
type CategoryInput struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	ParentID    string `json:"parent_id"`
	IsActive    bool   `json:"is_active"`
	SortOrder   int64  `json:"sort_order"`
	Image       *Image `json:"image"`
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
	Name           string      `json:"name"`
	Slug           string      `json:"slug"`
	CategoryID     string      `json:"category_id"`
	Description    string      `json:"description"`
	Material       string      `json:"material"`
	Price          int64       `json:"price"`
	CompareAtPrice *int64      `json:"compare_at_price"`
	Images         []string    `json:"images"`
	Image          *Image      `json:"image"`
	Gallery        []Image     `json:"gallery"`
	Colors         []Color     `json:"colors"`
	Accessories    []Accessory `json:"accessories"`
	IsActive       bool        `json:"is_active"`
}
type ProductCreate struct {
	ProductInput
	Variants []Variant `json:"variants"`
}
type InventoryDelta struct {
	Delta int64 `json:"delta"`
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
	CategoryExists(context.Context, string) (bool, error)
	CreateCategory(context.Context, CategoryInput) (map[string]any, error)
	UpdateCategory(context.Context, string, CategoryInput) (map[string]any, error)
	CreateProduct(context.Context, ProductCreate) (map[string]any, error)
	UpdateProduct(context.Context, string, ProductInput) (map[string]any, error)
	AddVariant(context.Context, string, Variant) (map[string]any, error)
	UpdateVariant(context.Context, string, string, VariantDetails) (map[string]any, error)
	AdjustInventory(context.Context, string, string, int64) (map[string]any, error)
}
type Service struct{ repo Repository }

func NewService(repo Repository) *Service { return &Service{repo: repo} }
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

func field(v, name string, limit int) error {
	v = strings.TrimSpace(v)
	if v == "" || !utf8.ValidString(v) || utf8.RuneCountInString(v) > limit || strings.ContainsAny(v, "\x00\r\n") {
		return Invalid{name, "Giá trị bắt buộc, không hợp lệ hoặc quá dài"}
	}
	return nil
}
func validateSlug(v string) error {
	if len(v) > 150 || !slugPattern.MatchString(v) {
		return Invalid{"slug", "Dùng chữ thường, số và dấu gạch ngang"}
	}
	return nil
}
func validateImage(i *Image, path string) error {
	if i == nil {
		return nil
	}
	if err := field(i.Src, path+".src", 2048); err != nil {
		return err
	}
	if err := field(i.Alt, path+".alt", 200); err != nil {
		return err
	}
	u, err := url.Parse(i.Src)
	if err != nil || u.User != nil || (u.Scheme != "https" && u.Scheme != "http" && !(u.Scheme == "" && strings.HasPrefix(i.Src, "/"))) {
		return Invalid{path + ".src", "Chỉ chấp nhận URL http(s) hoặc đường dẫn nội bộ"}
	}
	if u.Scheme == "" && u.Host != "" {
		return Invalid{path + ".src", "Đường dẫn nội bộ không được bắt đầu bằng //"}
	}
	if u.Scheme != "" && u.Host == "" {
		return Invalid{path + ".src", "Tên miền ảnh không hợp lệ"}
	}
	return nil
}
func validateCategory(c CategoryInput) error {
	if err := field(c.Name, "name", 150); err != nil {
		return err
	}
	if err := validateSlug(c.Slug); err != nil {
		return err
	}
	if utf8.RuneCountInString(c.Description) > 2000 {
		return Invalid{"description", "Mô tả quá dài"}
	}
	if c.SortOrder < 0 || c.SortOrder > 1000000 {
		return Invalid{"sort_order", "Thứ tự phải từ 0 đến 1000000"}
	}
	return validateImage(c.Image, "image")
}
func validateProduct(p ProductInput) error {
	if err := field(p.Name, "name", 200); err != nil {
		return err
	}
	if err := validateSlug(p.Slug); err != nil {
		return err
	}
	if err := field(p.CategoryID, "category_id", 24); err != nil {
		return err
	}
	if len(p.CategoryID) != 24 {
		return Invalid{"category_id", "Mã danh mục không hợp lệ"}
	}
	if p.Price < 0 || p.Price > math.MaxInt64/2 {
		return Invalid{"price", "Giá không hợp lệ"}
	}
	if p.CompareAtPrice != nil && *p.CompareAtPrice < 0 {
		return Invalid{"compare_at_price", "Giá không được âm"}
	}
	if utf8.RuneCountInString(p.Description) > 4000 {
		return Invalid{"description", "Mô tả quá dài"}
	}
	if utf8.RuneCountInString(p.Material) > 200 {
		return Invalid{"material", "Chất liệu quá dài"}
	}
	if len(p.Images) > 40 || len(p.Gallery) > 40 {
		return Invalid{"images", "Quá nhiều hình ảnh"}
	}
	if err := validateImage(p.Image, "image"); err != nil {
		return err
	}
	for i, x := range p.Gallery {
		if err := validateImage(&x, fmt.Sprintf("gallery[%d]", i)); err != nil {
			return err
		}
	}
	for i, s := range p.Images {
		if err := validateImage(&Image{Src: s, Alt: p.Name}, fmt.Sprintf("images[%d]", i)); err != nil {
			return err
		}
	}
	if len(p.Colors) > 100 || len(p.Accessories) > 40 {
		return Invalid{"colors", "Quá nhiều màu hoặc phụ kiện"}
	}
	for i, c := range p.Colors {
		if err := field(c.Name, fmt.Sprintf("colors[%d].name", i), 80); err != nil {
			return err
		}
		if !regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`).MatchString(c.Hex) {
			return Invalid{"colors.hex", "Mã HEX phải có dạng #RRGGBB"}
		}
	}
	for i, a := range p.Accessories {
		if err := field(a.ID, fmt.Sprintf("accessories[%d].id", i), 80); err != nil {
			return err
		}
		if err := field(a.Name, fmt.Sprintf("accessories[%d].name", i), 200); err != nil {
			return err
		}
		if a.Price < 0 {
			return Invalid{"accessories.price", "Giá phụ kiện không được âm"}
		}
		if err := validateImage(&a.Image, fmt.Sprintf("accessories[%d].image", i)); err != nil {
			return err
		}
	}
	return nil
}
func validateVariant(v Variant) error {
	if err := field(v.SKU, "sku", 80); err != nil {
		return err
	}
	if err := field(v.Color, "color", 80); err != nil {
		return err
	}
	if err := field(v.Size, "size", 40); err != nil {
		return err
	}
	if v.Stock < 0 || v.Stock > 1000000000 {
		return Invalid{"stock", "Tồn kho không hợp lệ"}
	}
	return nil
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
func (s *Service) CreateCategory(ctx context.Context, v CategoryInput) (map[string]any, error) {
	if err := validateCategory(v); err != nil {
		return nil, err
	}
	return s.repo.CreateCategory(ctx, v)
}
func (s *Service) UpdateCategory(ctx context.Context, id string, v CategoryInput) (map[string]any, error) {
	if err := validateCategory(v); err != nil {
		return nil, err
	}
	if id == v.ParentID {
		return nil, Invalid{"parent_id", "Danh mục không thể là cha của chính nó"}
	}
	return s.repo.UpdateCategory(ctx, id, v)
}
func (s *Service) CreateProduct(ctx context.Context, v ProductCreate) (map[string]any, error) {
	if err := validateProduct(v.ProductInput); err != nil {
		return nil, err
	}
	if err := s.checkCategory(ctx, v.CategoryID); err != nil {
		return nil, err
	}
	if len(v.Variants) < 1 || len(v.Variants) > 100 {
		return nil, Invalid{"variants", "Cần từ 1 đến 100 SKU"}
	}
	uniqueSKU := map[string]bool{}
	uniqueCombo := map[string]bool{}
	total := int64(0)
	for _, variant := range v.Variants {
		if err := validateVariant(variant); err != nil {
			return nil, err
		}
		key := strings.ToLower(variant.SKU)
		combo := strings.ToLower(variant.Color) + "\x00" + strings.ToLower(variant.Size)
		if uniqueSKU[key] || uniqueCombo[combo] {
			return nil, Invalid{"variants", "SKU hoặc tổ hợp màu/cỡ bị trùng"}
		}
		uniqueSKU[key] = true
		uniqueCombo[combo] = true
		if total > math.MaxInt64-variant.Stock {
			return nil, Invalid{"stock", "Tổng tồn kho quá lớn"}
		}
		total += variant.Stock
	}
	return s.repo.CreateProduct(ctx, v)
}
func (s *Service) UpdateProduct(ctx context.Context, id string, v ProductInput) (map[string]any, error) {
	if err := validateProduct(v); err != nil {
		return nil, err
	}
	if err := s.checkCategory(ctx, v.CategoryID); err != nil {
		return nil, err
	}
	return s.repo.UpdateProduct(ctx, id, v)
}
func (s *Service) AddVariant(ctx context.Context, id string, v Variant) (map[string]any, error) {
	// Newly added SKUs start with no stock; inventory must be changed through
	// an atomic delta so concurrent orders cannot be silently overwritten.
	if err := validateVariant(v); err != nil {
		return nil, err
	}
	if v.Stock != 0 {
		return nil, Invalid{"stock", "SKU mới bắt đầu với tồn kho bằng 0"}
	}
	return s.repo.AddVariant(ctx, id, v)
}
func (s *Service) UpdateVariant(ctx context.Context, id, sku string, v VariantDetails) (map[string]any, error) {
	if err := field(v.Color, "color", 80); err != nil {
		return nil, err
	}
	if err := field(v.Size, "size", 40); err != nil {
		return nil, err
	}
	return s.repo.UpdateVariant(ctx, id, sku, v)
}
func (s *Service) AdjustInventory(ctx context.Context, id, sku string, delta int64) (map[string]any, error) {
	if delta == 0 || delta < -1000000000 || delta > 1000000000 {
		return nil, Invalid{"delta", "Thay đổi tồn kho phải khác 0 và trong giới hạn cho phép"}
	}
	return s.repo.AdjustInventory(ctx, id, sku, delta)
}
