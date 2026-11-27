package catalogadmin

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
)

// imageURLs returns every image URL currently on a stored product map.
func storedImageURLs(m map[string]any) map[string]bool {
	out := map[string]bool{}
	add := func(v any) {
		if mm, ok := v.(map[string]any); ok {
			if s := mapString(mm, "src"); s != "" {
				out[s] = true
			}
		}
	}
	add(m["image"])
	for _, key := range []string{"gallery"} {
		if list, ok := m[key].([]any); ok {
			for _, x := range list {
				add(x)
			}
		}
	}
	if list, ok := m["images"].([]any); ok {
		for _, x := range list {
			if s, ok := x.(string); ok {
				out[s] = true
			}
		}
	}
	if list, ok := m["accessories"].([]any); ok {
		for _, x := range list {
			if a, ok := x.(map[string]any); ok {
				add(a["image"])
			}
		}
	}
	return out
}

// checkImageURL accepts a URL under the bucket prefix or one already stored on the record.
func (s *Service) checkImageURL(existing map[string]bool, path, src string) error {
	if s.imagePrefix == "" || src == "" || existing[src] || strings.HasPrefix(src, s.imagePrefix+"/") {
		return nil
	}
	return Invalid{path, "Ảnh phải được tải lên từ trang quản trị. Không dùng liên kết ảnh bên ngoài."}
}

func (s *Service) checkProductImages(existing map[string]bool, p ProductInput) error {
	if p.Image != nil {
		if err := s.checkImageURL(existing, "image.src", p.Image.Src); err != nil {
			return err
		}
	}
	for i, g := range p.Gallery {
		if err := s.checkImageURL(existing, fmt.Sprintf("gallery[%d].src", i), g.Src); err != nil {
			return err
		}
	}
	for i, src := range p.Images {
		if err := s.checkImageURL(existing, fmt.Sprintf("images[%d]", i), src); err != nil {
			return err
		}
	}
	for i, a := range p.Accessories {
		if a.Image != nil {
			if err := s.checkImageURL(existing, fmt.Sprintf("accessories[%d].image.src", i), a.Image.Src); err != nil {
				return err
			}
		}
	}
	return nil
}

func colorIndex(colors []string, name string) (string, bool) {
	for _, c := range colors {
		if strings.EqualFold(strings.TrimSpace(c), strings.TrimSpace(name)) {
			return c, true
		}
	}
	return "", false
}

func (s *Service) slugTaken(ctx context.Context, slug, exceptID string) (bool, error) {
	found, err := s.repo.ProductsBySlug(ctx, []string{slug})
	if err != nil {
		return false, err
	}
	for _, p := range found {
		if mapString(p, "id") != exceptID {
			return true, nil
		}
	}
	return false, nil
}

func (s *Service) CreateCategory(ctx context.Context, v CategoryInput) (map[string]any, error) {
	if err := validateCategory(&v); err != nil {
		return nil, err
	}
	list, err := s.repo.Categories(ctx)
	if err != nil {
		return nil, err
	}
	for _, c := range list {
		if mapString(c, "slug") == v.Slug {
			return nil, Invalid{"slug", "Đường dẫn này đã được dùng cho danh mục \"" + mapString(c, "name") + "\""}
		}
	}
	if v.Image != nil {
		if err := s.checkImageURL(nil, "image.src", v.Image.Src); err != nil {
			return nil, err
		}
	}
	out, err := s.repo.CreateCategory(ctx, v)
	if errors.Is(err, ErrConflict) {
		return nil, Invalid{"slug", "Đường dẫn này đã được dùng cho danh mục khác"}
	}
	return out, err
}

func (s *Service) UpdateCategory(ctx context.Context, id string, v CategoryInput) (map[string]any, error) {
	if err := validateCategory(&v); err != nil {
		return nil, err
	}
	if err := expectedVersion(v.ExpectedVersion); err != nil {
		return nil, err
	}
	list, err := s.repo.Categories(ctx)
	if err != nil {
		return nil, err
	}
	self := canonicalObjectID(id)
	parentOf := map[string]string{}
	var current map[string]any
	for _, c := range list {
		cid := canonicalObjectID(mapString(c, "id"))
		parentOf[cid] = canonicalObjectID(mapString(c, "parent_id"))
		if cid == self {
			current = c
		}
		if mapString(c, "slug") == v.Slug && cid != self {
			return nil, Invalid{"slug", "Đường dẫn này đã được dùng cho danh mục \"" + mapString(c, "name") + "\""}
		}
	}
	if self != "" {
		for cur, hops := canonicalObjectID(v.ParentID), 0; cur != ""; cur, hops = parentOf[cur], hops+1 {
			if cur == self {
				return nil, Invalid{"parent_id", "Danh mục cha không được là chính nó hoặc danh mục con của nó"}
			}
			if hops > len(list) {
				break
			}
		}
	}
	if v.Image != nil {
		existing := map[string]bool{}
		if current != nil {
			existing = storedImageURLs(current)
		}
		if err := s.checkImageURL(existing, "image.src", v.Image.Src); err != nil {
			return nil, err
		}
	}
	return s.repo.UpdateCategory(ctx, id, v)
}

// attributeProductConflict turns a duplicate-key conflict into an error on the offending field.
func (s *Service) attributeProductConflict(ctx context.Context, err error, slug string, variants []Variant) error {
	if !errors.Is(err, ErrConflict) {
		return err
	}
	if taken, e := s.slugTaken(ctx, slug, ""); e == nil && taken {
		return Invalid{"slug", "Đường dẫn này đã được dùng cho sản phẩm khác"}
	}
	skus := make([]string, 0, len(variants))
	for _, v := range variants {
		skus = append(skus, v.SKU)
	}
	if owners, e := s.repo.SKUOwners(ctx, skus); e == nil {
		for i, v := range variants {
			if owner, ok := owners[strings.ToLower(v.SKU)]; ok {
				return Invalid{fmt.Sprintf("variants[%d].sku", i), "SKU đã thuộc sản phẩm \"" + owner + "\""}
			}
		}
	}
	return err
}

func (s *Service) CreateProduct(ctx context.Context, v ProductCreate) (map[string]any, error) {
	if err := validateProduct(&v.ProductInput); err != nil {
		return nil, err
	}
	if err := s.checkCategory(ctx, v.CategoryID); err != nil {
		return nil, err
	}
	if taken, err := s.slugTaken(ctx, v.Slug, ""); err != nil {
		return nil, err
	} else if taken {
		return nil, Invalid{"slug", "Đường dẫn này đã được dùng cho sản phẩm khác"}
	}
	if len(v.Variants) < 1 || len(v.Variants) > 100 {
		return nil, Invalid{"variants", "Cần từ 1 đến 100 SKU"}
	}
	if len(v.Colors) == 0 {
		return nil, Invalid{"colors", "Thêm ít nhất một màu trước khi tạo SKU"}
	}
	colorNames := make([]string, 0, len(v.Colors))
	for _, c := range v.Colors {
		colorNames = append(colorNames, c.Name)
	}
	uniqueSKU := map[string]bool{}
	uniqueCombo := map[string]int{}
	skus := make([]string, 0, len(v.Variants))
	total := int64(0)
	for i := range v.Variants {
		variant := &v.Variants[i]
		prefix := fmt.Sprintf("variants[%d].", i)
		if err := validateVariant(variant); err != nil {
			return nil, withPath(prefix, err)
		}
		canonical, ok := colorIndex(colorNames, variant.Color)
		if !ok {
			return nil, Invalid{prefix + "color", "Màu phải chọn trong danh sách màu của sản phẩm"}
		}
		variant.Color = canonical
		if uniqueSKU[variant.SKU] {
			return nil, Invalid{prefix + "sku", "SKU bị trùng trong sản phẩm"}
		}
		uniqueSKU[variant.SKU] = true
		combo := strings.ToLower(variant.Color) + "\x00" + strings.ToLower(variant.Size)
		if prev, dup := uniqueCombo[combo]; dup {
			return nil, Invalid{prefix + "color", fmt.Sprintf("Cặp màu/cỡ bị trùng với SKU thứ %d", prev+1)}
		}
		uniqueCombo[combo] = i
		if total > math.MaxInt64-variant.Stock {
			return nil, Invalid{prefix + "stock", "Tổng tồn kho quá lớn"}
		}
		total += variant.Stock
		skus = append(skus, variant.SKU)
	}
	owners, err := s.repo.SKUOwners(ctx, skus)
	if err != nil {
		return nil, err
	}
	for i, variant := range v.Variants {
		if owner, taken := owners[strings.ToLower(variant.SKU)]; taken {
			return nil, Invalid{fmt.Sprintf("variants[%d].sku", i), "SKU đã thuộc sản phẩm \"" + owner + "\""}
		}
	}
	if v.IsActive && (v.Image == nil || strings.TrimSpace(v.Image.Src) == "") {
		return nil, Invalid{"is_active", "Cần có ảnh chính trước khi bật Đang bán"}
	}
	if err := s.checkProductImages(nil, v.ProductInput); err != nil {
		return nil, err
	}
	out, err := s.repo.CreateProduct(ctx, v)
	return out, s.attributeProductConflict(ctx, err, v.Slug, v.Variants)
}

func (s *Service) UpdateProduct(ctx context.Context, id string, v ProductInput) (map[string]any, error) {
	if err := validateProduct(&v); err != nil {
		return nil, err
	}
	if err := s.checkCategory(ctx, v.CategoryID); err != nil {
		return nil, err
	}
	if err := expectedVersion(v.ExpectedVersion); err != nil {
		return nil, err
	}
	state, err := s.repo.ProductState(ctx, id)
	if err != nil {
		return nil, err
	}
	if taken, err := s.slugTaken(ctx, v.Slug, state.ID); err != nil {
		return nil, err
	} else if taken {
		return nil, Invalid{"slug", "Đường dẫn này đã được dùng cho sản phẩm khác"}
	}
	// Turning a product on needs a main image and at least one SKU (products already on sale keep saving as before).
	if v.IsActive && !state.Active {
		if v.Image == nil || strings.TrimSpace(v.Image.Src) == "" {
			return nil, Invalid{"is_active", "Cần có ảnh chính trước khi bật Đang bán"}
		}
		if len(state.Variants) == 0 {
			return nil, Invalid{"is_active", "Cần có ít nhất 1 SKU trước khi bật Đang bán"}
		}
	}
	if len(state.Colors) > 0 {
		newColors := make([]string, 0, len(v.Colors))
		for i, c := range v.Colors {
			newColors = append(newColors, c.Name)
			_ = i
		}
		for _, variant := range state.Variants {
			if _, ok := colorIndex(newColors, variant.Color); !ok {
				return nil, Invalid{"colors", "Màu \"" + variant.Color + "\" đang được SKU " + variant.SKU + " sử dụng nên không xóa hoặc đổi tên được"}
			}
		}
	}
	existing := map[string]bool{}
	for _, u := range state.Images {
		existing[u] = true
	}
	if err := s.checkProductImages(existing, v); err != nil {
		return nil, err
	}
	return s.repo.UpdateProduct(ctx, id, v)
}

func (s *Service) AddVariant(ctx context.Context, id string, v Variant) (map[string]any, error) {
	// Newly added SKUs start with no stock; inventory must be changed through
	// an atomic delta so concurrent orders cannot be silently overwritten.
	if err := validateVariant(&v); err != nil {
		return nil, err
	}
	if v.Stock != 0 {
		return nil, Invalid{"stock", "SKU mới bắt đầu với tồn kho bằng 0"}
	}
	state, err := s.repo.ProductState(ctx, id)
	if err != nil {
		return nil, err
	}
	if len(state.Colors) > 0 {
		canonical, ok := colorIndex(state.Colors, v.Color)
		if !ok {
			return nil, Invalid{"color", "Màu phải chọn trong danh sách màu của sản phẩm"}
		}
		v.Color = canonical
	}
	if len(state.Variants) >= 100 {
		return nil, Invalid{"sku", "Mỗi sản phẩm tối đa 100 SKU"}
	}
	for _, other := range state.Variants {
		if strings.EqualFold(other.SKU, v.SKU) {
			return nil, Invalid{"sku", "SKU đã tồn tại trong sản phẩm này"}
		}
		if strings.EqualFold(other.Color, v.Color) && strings.EqualFold(other.Size, v.Size) {
			return nil, Invalid{"color", "Đã có SKU " + other.SKU + " cùng màu và cỡ"}
		}
	}
	owners, err := s.repo.SKUOwners(ctx, []string{v.SKU})
	if err != nil {
		return nil, err
	}
	if owner, taken := owners[strings.ToLower(v.SKU)]; taken {
		return nil, Invalid{"sku", "SKU đã thuộc sản phẩm \"" + owner + "\""}
	}
	out, err := s.repo.AddVariant(ctx, id, v)
	if errors.Is(err, ErrConflict) {
		return nil, Invalid{"sku", "SKU hoặc cặp màu/cỡ này đã tồn tại"}
	}
	return out, err
}

func (s *Service) UpdateVariant(ctx context.Context, id, sku string, v VariantDetails) (map[string]any, error) {
	if err := trimmedText(&v.Color, "color", "Màu", 1, colorNameMax); err != nil {
		return nil, err
	}
	size, err := canonicalSize(v.Size)
	if err != nil {
		return nil, err
	}
	v.Size = size
	state, err := s.repo.ProductState(ctx, id)
	if err != nil {
		return nil, err
	}
	if len(state.Colors) > 0 {
		canonical, ok := colorIndex(state.Colors, v.Color)
		if !ok {
			return nil, Invalid{"color", "Màu phải chọn trong danh sách màu của sản phẩm"}
		}
		v.Color = canonical
	}
	for _, other := range state.Variants {
		if !strings.EqualFold(other.SKU, sku) && strings.EqualFold(other.Color, v.Color) && strings.EqualFold(other.Size, v.Size) {
			return nil, Invalid{"color", "Đã có SKU " + other.SKU + " cùng màu và cỡ"}
		}
	}
	return s.repo.UpdateVariant(ctx, id, sku, v)
}

func (s *Service) AdjustInventory(ctx context.Context, id, sku string, delta int64, requestID string) (map[string]any, error) {
	if delta == 0 || delta < -stockMax || delta > stockMax {
		return nil, Invalid{"delta", fmt.Sprintf("Số điều chỉnh phải là số nguyên khác 0, tối đa %d", stockMax)}
	}
	// Retry identity is mandatory: a repeated network request must not apply
	// a second delta, even if the previous response was lost.
	if !uuidPattern.MatchString(requestID) {
		return nil, Invalid{"request_id", "Cần mã yêu cầu UUID hợp lệ"}
	}
	out, err := s.repo.AdjustInventory(ctx, id, sku, delta, strings.ToLower(requestID))
	if err != nil && delta < 0 && errors.Is(err, ErrConflict) {
		// Read after the fact so an idempotent retry of an applied delta is never mis-reported.
		if state, e := s.repo.ProductState(ctx, id); e == nil {
			for _, variant := range state.Variants {
				if strings.EqualFold(variant.SKU, sku) && variant.Stock+delta < 0 {
					return nil, Invalid{"delta", fmt.Sprintf("Không thể trừ %d: kho chỉ còn %d", -delta, variant.Stock)}
				}
			}
		}
	}
	return out, err
}
