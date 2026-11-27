package catalogadmin

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/your-org/aodora/server/internal/domain/catalog"
)

const (
	MaxImportRows = 5000
	MaxImportSize = 5 << 20
)

var (
	categoryColumns = []string{"name", "slug", "description", "parent_slug", "sort_order", "is_active", "image_url", "image_alt"}
	productColumns  = []string{"product_slug", "name", "category_slug", "description", "material", "price", "compare_at_price", "is_active", "image_url", "gallery_urls", "color_name", "color_hex", "size", "sku", "stock"}
)

// CategoryColumns / ProductColumns are the import (and export) headers.
func CategoryColumns() []string { return append([]string(nil), categoryColumns...) }
func ProductColumns() []string  { return append([]string(nil), productColumns...) }

// ImportError points at a row (1 = header row for CSV) and column.
type ImportError struct {
	Row     int    `json:"row"`
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ImportReport is returned by dry runs and commits.
type ImportReport struct {
	Mode         string        `json:"mode"`
	Created      int           `json:"created"`
	Updated      int           `json:"updated"`
	Skipped      int           `json:"skipped"`
	IgnoredStock int           `json:"ignored_stock"`
	Rows         int           `json:"rows"`
	Committed    bool          `json:"committed"`
	Errors       []ImportError `json:"errors"`
	Notes        []string      `json:"notes"`
	Items        []ImportItem  `json:"items"`
}

// ImportItem describes what happens to one category/product.
type ImportItem struct {
	Row    int    `json:"row"`
	Slug   string `json:"slug"`
	Action string `json:"action"` // create | update | skip
	Detail string `json:"detail,omitempty"`
}

// Row is one parsed input line with its original number.
type Row struct {
	Number int
	Values map[string]string
}

func (r Row) get(key string) string { return strings.TrimSpace(r.Values[key]) }

// ParseRows reads CSV (UTF-8, BOM tolerated) or a JSON array of flat objects.
func ParseRows(data []byte, format string, allowed []string, required []string) ([]Row, []ImportError) {
	if len(data) > MaxImportSize {
		return nil, []ImportError{{Row: 0, Field: "file", Message: fmt.Sprintf("File quá lớn (tối đa %d MB)", MaxImportSize>>20)}}
	}
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	var rows []Row
	var errs []ImportError
	switch format {
	case "json":
		rows, errs = parseJSONRows(data)
	default:
		rows, errs = parseCSVRows(data)
	}
	if len(errs) > 0 {
		return nil, errs
	}
	if len(rows) > MaxImportRows {
		return nil, []ImportError{{Row: 0, Field: "file", Message: fmt.Sprintf("Tối đa %d dòng mỗi lần nhập", MaxImportRows)}}
	}
	if len(rows) == 0 {
		return nil, []ImportError{{Row: 0, Field: "file", Message: "File không có dòng dữ liệu nào"}}
	}
	// Header check on the union of keys.
	seen := map[string]bool{}
	for _, r := range rows {
		for k := range r.Values {
			seen[k] = true
		}
	}
	allowedSet := map[string]bool{}
	for _, c := range allowed {
		allowedSet[c] = true
	}
	var headerErrs []ImportError
	for _, c := range required {
		if !seen[c] {
			headerErrs = append(headerErrs, ImportError{Row: 1, Field: c, Message: "Thiếu cột bắt buộc"})
		}
	}
	keys := make([]string, 0, len(seen))
	for k := range seen {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if !allowedSet[k] {
			headerErrs = append(headerErrs, ImportError{Row: 1, Field: k, Message: "Cột không được hỗ trợ"})
		}
	}
	if len(headerErrs) > 0 {
		return nil, headerErrs
	}
	return rows, nil
}

func parseCSVRows(data []byte) ([]Row, []ImportError) {
	reader := csv.NewReader(bytes.NewReader(data))
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true
	header, err := reader.Read()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, []ImportError{{Row: 1, Field: "file", Message: "File trống"}}
		}
		return nil, []ImportError{{Row: 1, Field: "file", Message: "Không đọc được CSV: " + err.Error()}}
	}
	for i := range header {
		header[i] = strings.ToLower(strings.TrimSpace(header[i]))
	}
	var rows []Row
	number := 1
	for {
		record, err := reader.Read()
		number++
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, []ImportError{{Row: number, Field: "file", Message: "Dòng CSV không hợp lệ: " + err.Error()}}
		}
		if len(record) > len(header) {
			return nil, []ImportError{{Row: number, Field: "file", Message: "Dòng có nhiều ô hơn số cột của tiêu đề"}}
		}
		values := map[string]string{}
		blank := true
		for i, name := range header {
			if name == "" {
				continue
			}
			v := ""
			if i < len(record) {
				v = strings.TrimSpace(record[i])
			}
			if v != "" {
				blank = false
			}
			values[name] = v
		}
		if !blank {
			rows = append(rows, Row{Number: number, Values: values})
		}
	}
	return rows, nil
}

func parseJSONRows(data []byte) ([]Row, []ImportError) {
	var objects []map[string]any
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&objects); err != nil {
		return nil, []ImportError{{Row: 0, Field: "file", Message: "JSON phải là một mảng các object phẳng: " + err.Error()}}
	}
	rows := make([]Row, 0, len(objects))
	for i, object := range objects {
		values := map[string]string{}
		for k, v := range object {
			key := strings.ToLower(strings.TrimSpace(k))
			switch x := v.(type) {
			case nil:
				values[key] = ""
			case string:
				values[key] = strings.TrimSpace(x)
			case json.Number:
				values[key] = x.String()
			case bool:
				values[key] = strconv.FormatBool(x)
			case []any:
				parts := make([]string, 0, len(x))
				for _, p := range x {
					parts = append(parts, strings.TrimSpace(fmt.Sprint(p)))
				}
				values[key] = strings.Join(parts, "|")
			default:
				return nil, []ImportError{{Row: i + 1, Field: key, Message: "Giá trị phải là chuỗi, số hoặc boolean"}}
			}
		}
		rows = append(rows, Row{Number: i + 1, Values: values})
	}
	return rows, nil
}

func parseBool(v string, def bool) (bool, bool) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "":
		return def, true
	case "true", "1", "yes", "y", "có", "co":
		return true, true
	case "false", "0", "no", "n", "không", "khong":
		return false, true
	}
	return false, false
}

func parseInt(v string) (int64, bool) {
	n, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
	return n, err == nil
}

func addErr(errs *[]ImportError, row int, field, msg string) {
	*errs = append(*errs, ImportError{Row: row, Field: field, Message: msg})
}

// invalidToErr converts a validation error from the shared validators into a row error.
func invalidToErr(errs *[]ImportError, row int, err error, fallbackField string) {
	var invalid Invalid
	if errors.As(err, &invalid) {
		addErr(errs, row, invalid.Field, invalid.Message)
		return
	}
	addErr(errs, row, fallbackField, err.Error())
}

func newRequestID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func mapString(m map[string]any, key string) string {
	if s, ok := m[key].(string); ok {
		return s
	}
	return ""
}
func mapInt(m map[string]any, key string) int64 {
	switch v := m[key].(type) {
	case int64:
		return v
	case int32:
		return int64(v)
	case int:
		return int64(v)
	case float64:
		return int64(math.Round(v))
	case json.Number:
		n, _ := v.Int64()
		return n
	}
	return 0
}

// ---------------------------------------------------------------- categories

type categoryPlan struct {
	row        Row
	input      CategoryInput
	parentSlug string
	existing   map[string]any
	action     string
}

func (s *Service) ImportCategories(ctx context.Context, rows []Row, commit bool) (ImportReport, error) {
	report := ImportReport{Mode: modeName(commit), Rows: len(rows), Errors: []ImportError{}, Notes: []string{}, Items: []ImportItem{}}
	existing, err := s.repo.Categories(ctx)
	if err != nil {
		return report, err
	}
	bySlug := map[string]map[string]any{}
	for _, c := range existing {
		bySlug[mapString(c, "slug")] = c
	}
	plans := map[string]*categoryPlan{}
	var order []*categoryPlan
	for _, row := range rows {
		p := &categoryPlan{row: row}
		in := &p.input
		in.Name, in.Slug, in.Description, p.parentSlug = row.get("name"), row.get("slug"), row.get("description"), row.get("parent_slug")
		if active, ok := parseBool(row.get("is_active"), true); ok {
			in.IsActive = active
		} else {
			addErr(&report.Errors, row.Number, "is_active", "Phải là true/false")
		}
		if v := row.get("sort_order"); v != "" {
			if n, ok := parseInt(v); ok {
				in.SortOrder = n
			} else {
				addErr(&report.Errors, row.Number, "sort_order", "Phải là số nguyên")
			}
		}
		if url := row.get("image_url"); url != "" {
			alt := row.get("image_alt")
			if alt == "" {
				alt = in.Name
			}
			in.Image = &Image{Src: url, Alt: alt}
		}
		if err := validateCategory(*in); err != nil {
			invalidToErr(&report.Errors, row.Number, err, "slug")
			continue
		}
		if prev, dup := plans[in.Slug]; dup {
			addErr(&report.Errors, row.Number, "slug", fmt.Sprintf("Slug trùng với dòng %d", prev.row.Number))
			continue
		}
		p.existing = bySlug[in.Slug]
		plans[in.Slug] = p
		order = append(order, p)
	}
	// Parent resolution and cycle detection over the combined (file overrides DB) graph.
	parentOf := map[string]string{}
	for slug, c := range bySlug {
		if pid := mapString(c, "parent_id"); pid != "" {
			for ps, pc := range bySlug {
				if mapString(pc, "id") == pid {
					parentOf[slug] = ps
				}
			}
		}
	}
	for _, p := range order {
		parentOf[p.input.Slug] = p.parentSlug
		if p.parentSlug == "" {
			continue
		}
		if p.parentSlug == p.input.Slug {
			addErr(&report.Errors, p.row.Number, "parent_slug", "Danh mục không thể là cha của chính nó")
			continue
		}
		if _, inFile := plans[p.parentSlug]; !inFile {
			if _, inDB := bySlug[p.parentSlug]; !inDB {
				addErr(&report.Errors, p.row.Number, "parent_slug", "Không tìm thấy danh mục cha \""+p.parentSlug+"\"")
			}
		}
	}
	for _, p := range order {
		seen := map[string]bool{p.input.Slug: true}
		for cur := parentOf[p.input.Slug]; cur != ""; cur = parentOf[cur] {
			if seen[cur] {
				addErr(&report.Errors, p.row.Number, "parent_slug", "Danh mục cha tạo thành vòng lặp")
				break
			}
			seen[cur] = true
		}
	}
	// Topological order: parents that are created in this file go first.
	sorted := make([]*categoryPlan, 0, len(order))
	placed := map[string]bool{}
	var place func(p *categoryPlan, depth int)
	place = func(p *categoryPlan, depth int) {
		if placed[p.input.Slug] || depth > len(order) {
			return
		}
		if parent, ok := plans[p.parentSlug]; ok && p.parentSlug != "" {
			place(parent, depth+1)
		}
		placed[p.input.Slug] = true
		sorted = append(sorted, p)
	}
	for _, p := range order {
		place(p, 0)
	}
	for _, p := range sorted {
		p.action = "create"
		if p.existing != nil {
			p.action = "update"
			if categoryUnchanged(p, bySlug) {
				p.action = "skip"
			}
		}
		report.Items = append(report.Items, ImportItem{Row: p.row.Number, Slug: p.input.Slug, Action: p.action})
		switch p.action {
		case "create":
			report.Created++
		case "update":
			report.Updated++
		default:
			report.Skipped++
		}
	}
	if len(report.Errors) > 0 || !commit {
		return report, nil
	}
	ids := map[string]string{}
	for slug, c := range bySlug {
		ids[slug] = mapString(c, "id")
	}
	written := 0
	for _, p := range sorted {
		p.input.ParentID = ids[p.parentSlug]
		var result map[string]any
		var err error
		switch p.action {
		case "create":
			result, err = s.CreateCategory(ctx, p.input)
		case "update":
			version := mapInt(p.existing, "version")
			p.input.ExpectedVersion = &version
			result, err = s.UpdateCategory(ctx, mapString(p.existing, "id"), p.input)
		default:
			continue
		}
		if err != nil {
			return failWritten(report, p.row.Number, "slug", err, written)
		}
		if id := mapString(result, "id"); id != "" {
			ids[p.input.Slug] = id
		}
		written++
	}
	report.Committed = true
	return report, nil
}

func categoryUnchanged(p *categoryPlan, bySlug map[string]map[string]any) bool {
	c := p.existing
	parentID := ""
	if p.parentSlug != "" {
		if parent, ok := bySlug[p.parentSlug]; ok {
			parentID = mapString(parent, "id")
		} else {
			return false // parent is created in this run
		}
	}
	var image *Image
	if m, ok := c["image"].(map[string]any); ok {
		image = &Image{Src: mapString(m, "src"), Alt: mapString(m, "alt")}
	}
	active, _ := c["is_active"].(bool)
	sameImage := (image == nil && p.input.Image == nil) || (image != nil && p.input.Image != nil && *image == *p.input.Image)
	return mapString(c, "name") == p.input.Name && mapString(c, "description") == p.input.Description && mapString(c, "parent_id") == parentID &&
		mapInt(c, "sort_order") == p.input.SortOrder && active == p.input.IsActive && sameImage
}

func modeName(commit bool) string {
	if commit {
		return "commit"
	}
	return "dry_run"
}

func failWritten(report ImportReport, row int, field string, err error, written int) (ImportReport, error) {
	var invalid Invalid
	switch {
	case errors.As(err, &invalid):
		addErr(&report.Errors, row, invalid.Field, invalid.Message)
	case errors.Is(err, ErrConflict):
		addErr(&report.Errors, row, field, "Dữ liệu đã thay đổi hoặc bị trùng, hãy chạy lại bước xem trước")
	case errors.Is(err, ErrTransactionsRequired):
		addErr(&report.Errors, row, field, "Đổi danh mục cha cần MongoDB replica set")
	default:
		return report, err
	}
	report.Committed = false
	report.Notes = append(report.Notes, fmt.Sprintf("Đã ghi %d mục trước khi dừng ở dòng %d; các mục này giữ nguyên, chạy lại sau khi sửa lỗi.", written, row))
	return report, nil
}

// ------------------------------------------------------------------ products

type variantPlan struct {
	row   int
	input Variant
}

type productPlan struct {
	slug     string
	first    int
	input    ProductInput
	hasImage bool
	hasColor bool
	colors   []Color
	variants []variantPlan
	existing map[string]any
	action   string
	detail   []string
	// differences against an existing product
	metadataChanged bool
	newVariants     []variantPlan
	editVariants    []variantPlan
}

func (s *Service) ImportProducts(ctx context.Context, rows []Row, commit bool) (ImportReport, error) {
	report := ImportReport{Mode: modeName(commit), Rows: len(rows), Errors: []ImportError{}, Notes: []string{}, Items: []ImportItem{}}
	categories, err := s.repo.Categories(ctx)
	if err != nil {
		return report, err
	}
	categoryID := map[string]string{}
	for _, c := range categories {
		categoryID[mapString(c, "slug")] = mapString(c, "id")
	}
	plans := map[string]*productPlan{}
	var order []*productPlan
	firstRow := map[string]Row{}
	skuRow := map[string]int{}
	for _, row := range rows {
		slug := row.get("product_slug")
		if slug == "" {
			addErr(&report.Errors, row.Number, "product_slug", "Giá trị bắt buộc")
			continue
		}
		p := plans[slug]
		isFirst := p == nil
		if isFirst {
			p = &productPlan{slug: slug, first: row.Number}
			plans[slug] = p
			firstRow[slug] = row
			order = append(order, p)
			s.fillProductFields(&report, p, row, categoryID)
		} else {
			compareProductFields(&report, firstRow[slug], row)
		}
		// variant
		v := Variant{SKU: row.get("sku"), Color: row.get("color_name")}
		if sizeRaw := row.get("size"); sizeRaw != "" {
			if canonical, ok := catalog.CanonicalSize(sizeRaw); ok {
				v.Size = canonical
			} else {
				addErr(&report.Errors, row.Number, "size", catalog.SizeMessage)
				continue
			}
		}
		if raw := row.get("stock"); raw != "" {
			n, ok := parseInt(raw)
			if !ok || n < 0 {
				addErr(&report.Errors, row.Number, "stock", "Tồn kho phải là số nguyên không âm")
				continue
			}
			v.Stock = n
		}
		if err := validateVariant(&v); err != nil {
			invalidToErr(&report.Errors, row.Number, err, "sku")
			continue
		}
		if prev, dup := skuRow[strings.ToLower(v.SKU)]; dup {
			addErr(&report.Errors, row.Number, "sku", fmt.Sprintf("SKU trùng với dòng %d", prev))
			continue
		}
		skuRow[strings.ToLower(v.SKU)] = row.Number
		if c := row.get("color_name"); c != "" {
			if hex := row.get("color_hex"); hex != "" && !validHex(hex) {
				addErr(&report.Errors, row.Number, "color_hex", "Mã màu phải có dạng #RRGGBB")
				continue
			} else {
				p.addColor(&report, row.Number, c, hex)
			}
		}
		p.variants = append(p.variants, variantPlan{row: row.Number, input: v})
	}
	// Look up what exists.
	slugs := make([]string, 0, len(order))
	skus := make([]string, 0, len(skuRow))
	for _, p := range order {
		slugs = append(slugs, p.slug)
	}
	for _, p := range order {
		for _, v := range p.variants {
			skus = append(skus, v.input.SKU)
		}
	}
	existing, err := s.repo.ProductsBySlug(ctx, slugs)
	if err != nil {
		return report, err
	}
	for _, e := range existing {
		if p := plans[mapString(e, "slug")]; p != nil {
			p.existing = e
		}
	}
	owners, err := s.repo.SKUOwners(ctx, skus)
	if err != nil {
		return report, err
	}
	for _, p := range order {
		s.finishProduct(&report, p, owners)
	}
	if len(report.Errors) == 0 {
		for _, p := range order {
			report.Items = append(report.Items, ImportItem{Row: p.first, Slug: p.slug, Action: p.action, Detail: strings.Join(p.detail, "; ")})
			switch p.action {
			case "create":
				report.Created++
			case "update":
				report.Updated++
			default:
				report.Skipped++
			}
		}
	}
	if report.IgnoredStock > 0 {
		report.Notes = append(report.Notes, fmt.Sprintf("%d SKU đã có: tồn kho trong file bị bỏ qua (tồn chỉ đổi bằng điều chỉnh +/- trong trang sản phẩm).", report.IgnoredStock))
	}
	if len(report.Errors) > 0 || !commit {
		return report, nil
	}
	written := 0
	for _, p := range order {
		if p.action == "skip" {
			continue
		}
		if err := s.commitProduct(ctx, p); err != nil {
			return failWritten(report, p.first, "product_slug", err, written)
		}
		written++
	}
	report.Committed = true
	return report, nil
}

func validHex(v string) bool {
	if len(v) != 7 || v[0] != '#' {
		return false
	}
	for _, c := range v[1:] {
		if !strings.ContainsRune("0123456789abcdefABCDEF", c) {
			return false
		}
	}
	return true
}

func (p *productPlan) addColor(report *ImportReport, row int, name, hex string) {
	for i, c := range p.colors {
		if strings.EqualFold(c.Name, name) {
			if hex != "" && c.Hex != "" && !strings.EqualFold(c.Hex, hex) {
				addErr(&report.Errors, row, "color_hex", "Màu \""+name+"\" có mã khác ở dòng khác")
			} else if c.Hex == "" && hex != "" {
				p.colors[i].Hex = hex
			}
			return
		}
	}
	p.colors = append(p.colors, Color{Name: name, Hex: hex})
	p.hasColor = true
}

func splitGallery(v string) []string {
	var out []string
	for part := range strings.SplitSeq(v, "|") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}

func (s *Service) fillProductFields(report *ImportReport, p *productPlan, row Row, categoryID map[string]string) {
	before := len(report.Errors)
	in := &p.input
	in.Name, in.Slug, in.Description, in.Material = row.get("name"), p.slug, row.get("description"), row.get("material")
	if cat := row.get("category_slug"); cat == "" {
		addErr(&report.Errors, row.Number, "category_slug", "Giá trị bắt buộc")
	} else if id, ok := categoryID[cat]; ok {
		in.CategoryID = id
	} else {
		addErr(&report.Errors, row.Number, "category_slug", "Không tìm thấy danh mục \""+cat+"\" (nhập danh mục trước)")
	}
	if raw := row.get("price"); raw == "" {
		addErr(&report.Errors, row.Number, "price", "Giá trị bắt buộc")
	} else if n, ok := parseInt(raw); ok && n >= 0 {
		in.Price = n
	} else {
		addErr(&report.Errors, row.Number, "price", "Giá phải là số nguyên VND không âm, ví dụ 1500000")
	}
	if raw := row.get("compare_at_price"); raw != "" {
		if n, ok := parseInt(raw); ok && n >= 0 {
			in.CompareAtPrice = &n
		} else {
			addErr(&report.Errors, row.Number, "compare_at_price", "Giá gốc phải là số nguyên VND không âm")
		}
	}
	if active, ok := parseBool(row.get("is_active"), true); ok {
		in.IsActive = active
	} else {
		addErr(&report.Errors, row.Number, "is_active", "Phải là true/false")
	}
	if url := row.get("image_url"); url != "" {
		in.Image = &Image{Src: url, Alt: in.Name}
		p.hasImage = true
	}
	for _, g := range splitGallery(row.get("gallery_urls")) {
		in.Gallery = append(in.Gallery, Image{Src: g, Alt: in.Name})
		p.hasImage = true
	}
	if err := validateProduct(*in); err != nil && len(report.Errors) == before {
		invalidToErr(&report.Errors, row.Number, err, "name")
	}
}

// compareProductFields: later rows of a product may leave product-level columns blank
// but must not contradict the first row.
func compareProductFields(report *ImportReport, first, row Row) {
	for _, col := range []string{"name", "category_slug", "description", "material", "price", "compare_at_price", "is_active", "image_url", "gallery_urls"} {
		if v := row.get(col); v != "" && v != first.get(col) {
			addErr(&report.Errors, row.Number, col, fmt.Sprintf("Khác với dòng %d của cùng sản phẩm", first.Number))
		}
	}
}

func (s *Service) finishProduct(report *ImportReport, p *productPlan, owners map[string]string) {
	if len(p.variants) == 0 {
		return
	}
	for _, v := range p.variants {
		if owner, taken := owners[strings.ToLower(v.input.SKU)]; taken && owner != p.slug {
			addErr(&report.Errors, v.row, "sku", "SKU đã thuộc sản phẩm \""+owner+"\"")
		}
	}
	p.input.Colors = p.colors
	p.input.Images = nil
	if p.input.Image != nil {
		p.input.Images = append(p.input.Images, p.input.Image.Src)
	}
	for _, g := range p.input.Gallery {
		p.input.Images = append(p.input.Images, g.Src)
	}
	if p.existing == nil {
		p.action = "create"
		combo := map[string]int{}
		for _, v := range p.variants {
			key := strings.ToLower(v.input.Color) + "\x00" + strings.ToLower(v.input.Size)
			if prev, dup := combo[key]; dup {
				addErr(&report.Errors, v.row, "color_name", fmt.Sprintf("Tổ hợp màu/cỡ trùng với dòng %d", prev))
			}
			combo[key] = v.row
		}
		if len(p.variants) > 100 {
			addErr(&report.Errors, p.first, "sku", "Tối đa 100 SKU cho mỗi sản phẩm")
		}
		p.detail = append(p.detail, fmt.Sprintf("%d SKU mới", len(p.variants)))
		return
	}
	// existing product: merge, never wipe fields the file left blank
	var merged ProductInput
	raw, _ := json.Marshal(p.existing)
	_ = json.Unmarshal(raw, &merged)
	old := merged
	merged.Name, merged.CategoryID, merged.Price, merged.CompareAtPrice, merged.IsActive = p.input.Name, p.input.CategoryID, p.input.Price, p.input.CompareAtPrice, p.input.IsActive
	if p.input.Description != "" {
		merged.Description = p.input.Description
	}
	if p.input.Material != "" {
		merged.Material = p.input.Material
	}
	if p.hasImage {
		merged.Image, merged.Gallery, merged.Images = p.input.Image, p.input.Gallery, p.input.Images
	}
	if p.hasColor {
		merged.Colors = p.colors
	}
	version := mapInt(p.existing, "version")
	merged.ExpectedVersion = &version
	if err := validateProduct(merged); err != nil {
		invalidToErr(&report.Errors, p.first, err, "name")
	}
	old.ExpectedVersion = &version
	emptyNotNil(&old)
	emptyNotNil(&merged)
	a, _ := json.Marshal(old)
	b, _ := json.Marshal(merged)
	p.metadataChanged = !bytes.Equal(a, b)
	p.input = merged
	existingVariants := map[string]map[string]any{}
	combos := map[string]string{}
	if list, ok := p.existing["variants"].([]any); ok {
		for _, item := range list {
			if m, ok := item.(map[string]any); ok {
				sku := strings.ToLower(mapString(m, "sku"))
				existingVariants[sku] = m
				combos[strings.ToLower(mapString(m, "color"))+"\x00"+strings.ToLower(mapString(m, "size"))] = sku
			}
		}
	}
	for _, v := range p.variants {
		key := strings.ToLower(v.input.SKU)
		if e, ok := existingVariants[key]; ok {
			if mapString(e, "color") != v.input.Color || mapString(e, "size") != v.input.Size {
				p.editVariants = append(p.editVariants, v)
			}
			report.IgnoredStock++
			continue
		}
		combo := strings.ToLower(v.input.Color) + "\x00" + strings.ToLower(v.input.Size)
		if other, clash := combos[combo]; clash && other != key {
			addErr(&report.Errors, v.row, "color_name", "Tổ hợp màu/cỡ đã có ở SKU "+other)
		}
		combos[combo] = key
		p.newVariants = append(p.newVariants, v)
	}
	if len(existingVariants)+len(p.newVariants) > 100 {
		addErr(&report.Errors, p.first, "sku", "Tối đa 100 SKU cho mỗi sản phẩm")
	}
	switch {
	case p.metadataChanged || len(p.newVariants) > 0 || len(p.editVariants) > 0:
		p.action = "update"
		if p.metadataChanged {
			p.detail = append(p.detail, "cập nhật thông tin")
		}
		if len(p.newVariants) > 0 {
			p.detail = append(p.detail, fmt.Sprintf("%d SKU mới", len(p.newVariants)))
		}
		if len(p.editVariants) > 0 {
			p.detail = append(p.detail, fmt.Sprintf("sửa màu/cỡ %d SKU", len(p.editVariants)))
		}
	default:
		p.action = "skip"
	}
}

// emptyNotNil makes nil and empty slices compare equal (existing documents store [], the importer builds nil).
func emptyNotNil(p *ProductInput) {
	if p.Gallery == nil {
		p.Gallery = []Image{}
	}
	if p.Images == nil {
		p.Images = []string{}
	}
	if p.Colors == nil {
		p.Colors = []Color{}
	}
	if p.Accessories == nil {
		p.Accessories = []Accessory{}
	}
}

func (s *Service) commitProduct(ctx context.Context, p *productPlan) error {
	if p.action == "create" {
		variants := make([]Variant, 0, len(p.variants))
		for _, v := range p.variants {
			variants = append(variants, v.input)
		}
		_, err := s.CreateProduct(ctx, ProductCreate{ProductInput: p.input, Variants: variants})
		return err
	}
	id := mapString(p.existing, "id")
	if p.metadataChanged {
		if _, err := s.UpdateProduct(ctx, id, p.input); err != nil {
			return err
		}
	}
	for _, v := range p.editVariants {
		if _, err := s.UpdateVariant(ctx, id, v.input.SKU, VariantDetails{Color: v.input.Color, Size: v.input.Size}); err != nil {
			return err
		}
	}
	for _, v := range p.newVariants {
		stock := v.input.Stock
		v.input.Stock = 0 // new SKUs start at 0; the opening stock is applied as a delta
		if _, err := s.AddVariant(ctx, id, v.input); err != nil {
			return err
		}
		if stock > 0 {
			if _, err := s.AdjustInventory(ctx, id, v.input.SKU, stock, newRequestID()); err != nil {
				return err
			}
		}
	}
	return nil
}

// ------------------------------------------------------------------- export

// ExportCategories returns header + rows in the import format.
func (s *Service) ExportCategories(ctx context.Context) ([][]string, error) {
	list, err := s.repo.Categories(ctx)
	if err != nil {
		return nil, err
	}
	slugByID := map[string]string{}
	for _, c := range list {
		slugByID[mapString(c, "id")] = mapString(c, "slug")
	}
	out := [][]string{CategoryColumns()}
	for _, c := range list {
		imageURL, imageAlt := "", ""
		if m, ok := c["image"].(map[string]any); ok {
			imageURL, imageAlt = mapString(m, "src"), mapString(m, "alt")
		}
		active, _ := c["is_active"].(bool)
		out = append(out, []string{mapString(c, "name"), mapString(c, "slug"), mapString(c, "description"), slugByID[mapString(c, "parent_id")],
			strconv.FormatInt(mapInt(c, "sort_order"), 10), strconv.FormatBool(active), imageURL, imageAlt})
	}
	return out, nil
}

// ExportProducts returns one row per SKU (product columns repeated on every row).
func (s *Service) ExportProducts(ctx context.Context) ([][]string, error) {
	categories, err := s.repo.Categories(ctx)
	if err != nil {
		return nil, err
	}
	slugByID := map[string]string{}
	for _, c := range categories {
		slugByID[mapString(c, "id")] = mapString(c, "slug")
	}
	out := [][]string{ProductColumns()}
	for page := int64(1); page <= 100; page++ {
		result, err := s.repo.Products(ctx, page, 100)
		if err != nil {
			return nil, err
		}
		for _, p := range result.Items {
			imageURL := ""
			if m, ok := p["image"].(map[string]any); ok {
				imageURL = mapString(m, "src")
			}
			var gallery []string
			if list, ok := p["gallery"].([]any); ok {
				for _, g := range list {
					if m, ok := g.(map[string]any); ok {
						gallery = append(gallery, mapString(m, "src"))
					}
				}
			}
			hexByColor := map[string]string{}
			if list, ok := p["colors"].([]any); ok {
				for _, c := range list {
					if m, ok := c.(map[string]any); ok {
						hexByColor[strings.ToLower(mapString(m, "name"))] = mapString(m, "hex")
					}
				}
			}
			compare := ""
			if n := mapInt(p, "compare_at_price"); n > 0 {
				compare = strconv.FormatInt(n, 10)
			}
			active, _ := p["is_active"].(bool)
			list, _ := p["variants"].([]any)
			for _, item := range list {
				v, ok := item.(map[string]any)
				if !ok {
					continue
				}
				out = append(out, []string{
					mapString(p, "slug"), mapString(p, "name"), slugByID[mapString(p, "category_id")], mapString(p, "description"), mapString(p, "material"),
					strconv.FormatInt(mapInt(p, "price"), 10), compare, strconv.FormatBool(active), imageURL, strings.Join(gallery, "|"),
					mapString(v, "color"), hexByColor[strings.ToLower(mapString(v, "color"))], mapString(v, "size"), mapString(v, "sku"), strconv.FormatInt(mapInt(v, "stock"), 10),
				})
			}
		}
		if page >= result.TotalPages {
			break
		}
	}
	return out, nil
}
