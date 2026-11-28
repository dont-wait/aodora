package catalogadmin

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/your-org/aodora/server/internal/domain/catalog"
)

// Limits shared with the admin UI (client/src/features/admin/validation.ts must match).
const (
	nameMin, nameMax        = 3, 150
	slugMax                 = 120
	descriptionMax          = 5000
	materialMax             = 200
	priceMin, priceMax      = 1000, 1_000_000_000
	galleryMax              = 12
	altMax                  = 200
	colorsMax, colorNameMax = 20, 50
	accessoriesMax          = 20
	accessoryNameMax        = 100
	accessoryDescriptionMax = 500
	accessoryPriceMax       = 100_000_000
	accessoryTagMax         = 30
	stockMax                = 100_000
	categoryNameMin         = 2
	categoryNameMax         = 100
	categoryDescriptionMax  = 2000
	categorySortMax         = 9999
)

var (
	slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	skuPattern  = regexp.MustCompile(`^[A-Z0-9][A-Z0-9-]{2,39}$`)
	hexPattern  = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)
)

// field is the generic required-text check (non-empty, valid UTF-8, within limit, no control characters).
func field(v, name string, limit int) error {
	v = strings.TrimSpace(v)
	if v == "" || !utf8.ValidString(v) || utf8.RuneCountInString(v) > limit || strings.ContainsAny(v, "\x00\r\n") {
		return Invalid{name, "Giá trị bắt buộc, không hợp lệ hoặc quá dài"}
	}
	return nil
}

// trimmedText trims *v and checks its length in characters.
func trimmedText(v *string, name, label string, min, max int) error {
	*v = strings.TrimSpace(*v)
	n := utf8.RuneCountInString(*v)
	if !utf8.ValidString(*v) || strings.ContainsAny(*v, "\x00\r\n") {
		return Invalid{name, label + " chứa ký tự không hợp lệ"}
	}
	if n < min || n > max {
		if min <= 1 {
			return Invalid{name, fmt.Sprintf("%s bắt buộc, tối đa %d ký tự", label, max)}
		}
		return Invalid{name, fmt.Sprintf("%s phải từ %d đến %d ký tự", label, min, max)}
	}
	return nil
}

func validateSlug(v string) error {
	if v == "" || len(v) > slugMax || !slugPattern.MatchString(v) {
		return Invalid{"slug", fmt.Sprintf("Đường dẫn chỉ gồm chữ thường, số và dấu gạch ngang, tối đa %d ký tự", slugMax)}
	}
	return nil
}

func validateImage(i *Image, path string) error {
	if i == nil {
		return nil
	}
	if err := field(i.Src, path+".src", 2048); err != nil {
		return Invalid{path + ".src", "Ảnh chưa có đường dẫn hợp lệ"}
	}
	if err := trimmedText(&i.Alt, path+".alt", "Mô tả ảnh", 1, altMax); err != nil {
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

func validateCategory(c *CategoryInput) error {
	if err := trimmedText(&c.Name, "name", "Tên danh mục", categoryNameMin, categoryNameMax); err != nil {
		return err
	}
	if err := validateSlug(c.Slug); err != nil {
		return err
	}
	if utf8.RuneCountInString(c.Description) > categoryDescriptionMax {
		return Invalid{"description", fmt.Sprintf("Mô tả tối đa %d ký tự", categoryDescriptionMax)}
	}
	if c.SortOrder < 0 || c.SortOrder > categorySortMax {
		return Invalid{"sort_order", fmt.Sprintf("Thứ tự phải từ 0 đến %d", categorySortMax)}
	}
	return validateImage(c.Image, "image")
}

func validateProduct(p *ProductInput) error {
	if err := trimmedText(&p.Name, "name", "Tên sản phẩm", nameMin, nameMax); err != nil {
		return err
	}
	if err := validateSlug(p.Slug); err != nil {
		return err
	}
	if err := field(p.CategoryID, "category_id", 24); err != nil || len(p.CategoryID) != 24 {
		return Invalid{"category_id", "Chọn danh mục cho sản phẩm"}
	}
	if p.Price < priceMin || p.Price > priceMax {
		return Invalid{"price", fmt.Sprintf("Giá bán phải từ %d đến %d VND", priceMin, priceMax)}
	}
	if p.CompareAtPrice != nil && (*p.CompareAtPrice <= p.Price || *p.CompareAtPrice > priceMax) {
		return Invalid{"compare_at_price", "Giá gốc phải lớn hơn giá bán (hoặc để trống)"}
	}
	if utf8.RuneCountInString(p.Description) > descriptionMax {
		return Invalid{"description", fmt.Sprintf("Mô tả tối đa %d ký tự", descriptionMax)}
	}
	if utf8.RuneCountInString(p.Material) > materialMax {
		return Invalid{"material", fmt.Sprintf("Chất liệu tối đa %d ký tự", materialMax)}
	}
	if len(p.Gallery) > galleryMax {
		return Invalid{"gallery", fmt.Sprintf("Tối đa %d ảnh phụ", galleryMax)}
	}
	if len(p.Images) > galleryMax+1 {
		return Invalid{"images", "Quá nhiều hình ảnh"}
	}
	if err := validateImage(p.Image, "image"); err != nil {
		return err
	}
	// The same image may appear as main image and in the gallery (common in imports); the storefront shows it once.
	for i := range p.Gallery {
		if err := validateImage(&p.Gallery[i], fmt.Sprintf("gallery[%d]", i)); err != nil {
			return err
		}
	}
	for i, s := range p.Images {
		img := Image{Src: s, Alt: p.Name}
		if err := validateImage(&img, fmt.Sprintf("images[%d]", i)); err != nil {
			return err
		}
	}
	if len(p.Colors) > colorsMax {
		return Invalid{"colors", fmt.Sprintf("Tối đa %d màu", colorsMax)}
	}
	names := map[string]bool{}
	for i := range p.Colors {
		c := &p.Colors[i]
		if err := trimmedText(&c.Name, fmt.Sprintf("colors[%d].name", i), "Tên màu", 1, colorNameMax); err != nil {
			return err
		}
		key := foldText(c.Name)
		if names[key] {
			return Invalid{fmt.Sprintf("colors[%d].name", i), "Tên màu bị trùng"}
		}
		names[key] = true
		if c.Hex != "" {
			if !hexPattern.MatchString(c.Hex) {
				return Invalid{fmt.Sprintf("colors[%d].hex", i), "Mã màu phải có dạng #RRGGBB"}
			}
			c.Hex = strings.ToUpper(c.Hex)
		}
	}
	if len(p.Accessories) > accessoriesMax {
		return Invalid{"accessories", fmt.Sprintf("Tối đa %d phụ kiện", accessoriesMax)}
	}
	ids := map[string]bool{}
	for i := range p.Accessories {
		a := &p.Accessories[i]
		path := fmt.Sprintf("accessories[%d]", i)
		if err := field(a.ID, path+".id", 80); err != nil {
			return err
		}
		if ids[a.ID] {
			return Invalid{path + ".id", "Mã phụ kiện bị trùng"}
		}
		ids[a.ID] = true
		if err := trimmedText(&a.Name, path+".name", "Tên phụ kiện", 1, accessoryNameMax); err != nil {
			return err
		}
		if utf8.RuneCountInString(a.Description) > accessoryDescriptionMax {
			return Invalid{path + ".description", fmt.Sprintf("Mô tả phụ kiện tối đa %d ký tự", accessoryDescriptionMax)}
		}
		if a.Price < 0 || a.Price > accessoryPriceMax {
			return Invalid{path + ".price", fmt.Sprintf("Giá phụ kiện phải từ 0 đến %d VND", accessoryPriceMax)}
		}
		if utf8.RuneCountInString(a.Tag) > accessoryTagMax {
			return Invalid{path + ".tag", fmt.Sprintf("Nhãn tối đa %d ký tự", accessoryTagMax)}
		}
		if err := validateImage(a.Image, path+".image"); err != nil {
			return err
		}
	}
	return nil
}

// canonicalSize validates size against the standard list and returns its stored spelling.
func canonicalSize(size string) (string, error) {
	canonical, ok := catalog.CanonicalSize(size)
	if !ok {
		return "", Invalid{"size", catalog.SizeMessage}
	}
	return canonical, nil
}

// validateVariant normalises sku (upper-case, no diacritics) and size, and checks colour text and opening stock.
func validateVariant(v *Variant) error {
	v.SKU = normalizeSKU(v.SKU)
	if !skuPattern.MatchString(v.SKU) {
		return Invalid{"sku", "SKU gồm 3–40 ký tự: chữ in hoa, số và dấu gạch ngang, bắt đầu bằng chữ hoặc số"}
	}
	if err := trimmedText(&v.Color, "color", "Màu", 1, colorNameMax); err != nil {
		return err
	}
	size, err := canonicalSize(v.Size)
	if err != nil {
		return err
	}
	v.Size = size
	if v.Stock < 0 || v.Stock > stockMax {
		return Invalid{"stock", fmt.Sprintf("Tồn kho phải là số nguyên từ 0 đến %d", stockMax)}
	}
	return nil
}

// withPath prefixes the field of an Invalid error: ("variants[2].", sku) -> variants[2].sku.
func withPath(prefix string, err error) error {
	if invalid, ok := err.(Invalid); ok {
		return Invalid{prefix + invalid.Field, invalid.Message}
	}
	return err
}
