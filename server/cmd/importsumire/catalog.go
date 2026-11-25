package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"html"
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"go.mongodb.org/mongo-driver/v2/bson"
)

const (
	seedSource      = "sumire-store"
	imagePrefix     = "catalog/sumire/"
	slugPrefix      = "sumire-"
	collectionName  = "Sumire Store"
	discontinuedTag = "vd-ngung-sx"
	skippedType     = "Khác"
	maxDescription  = 1000
	fallbackHex     = "#CCCCCC"
)

type srcOption struct {
	Name     string   `json:"name"`
	Position int      `json:"position"`
	Values   []string `json:"values"`
}

type srcVariant struct {
	SKU            string  `json:"sku"`
	Option1        *string `json:"option1"`
	Option2        *string `json:"option2"`
	Option3        *string `json:"option3"`
	Price          string  `json:"price"`
	CompareAtPrice *string `json:"compare_at_price"`
	Available      bool    `json:"available"`
}

type srcImage struct {
	Order  int    `json:"order"`
	File   string `json:"file"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Source string `json:"source"`
}

type srcProduct struct {
	Handle      string       `json:"handle"`
	Title       string       `json:"title"`
	ProductType string       `json:"product_type"`
	Vendor      string       `json:"vendor"`
	Tags        []string     `json:"tags"`
	BodyHTML    string       `json:"body_html"`
	SourceURL   string       `json:"source_url"`
	Options     []srcOption  `json:"options"`
	Variants    []srcVariant `json:"variants"`
	Images      []srcImage   `json:"images"`
}

type imageRef struct{ Handle, File string }

type skipped struct{ Handle, Reason string }

type plan struct {
	Categories []bson.M
	Products   []bson.M
	Images     []imageRef
	Skipped    []skipped
	Warnings   []string
}

type categoryDef struct{ Key, Name string }

var (
	catTaDoi   = categoryDef{"ao-dai-ta-doi", "Áo dài tà đôi"}
	catTayChen = categoryDef{"ao-dai-tay-chen", "Áo dài tay chèn"}
	catTet     = categoryDef{"ao-dai-tet", "Áo dài Tết"}
	catDefault = categoryDef{"ao-dai-sumire", "Áo dài Sumire"}
	categories = []categoryDef{catTaDoi, catTayChen, catTet, catDefault}
)

var (
	handleRE     = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	blockTagRE   = regexp.MustCompile(`(?i)<br\s*/?>|</p>|</div>|</li>|</h[1-6]>`)
	anyTagRE     = regexp.MustCompile(`<[^>]*>`)
	separatorRE  = regexp.MustCompile(`^[\s_\-–—=.]+$`)
	materialRE   = regexp.MustCompile(`(?i)chất liệu\s*:\s*([^\n.,;:()]{2,40})`)
	sectionMatRE = regexp.MustCompile(`(?i)\[chất liệu\]\s*\n([^\n.,;:()]{2,40})`)
	sizePrefixRE = regexp.MustCompile(`(?i)^size\s+`)
)

// materialRE only trusts an explicit "Chất liệu: X" label; prose such as "chất liệu tơ tencel óng ả càng làm…" is ignored.
// boilerplate line prefixes copied from every Sumire description.
var boilerplate = []string{"tên sản phẩm", "chị bạn khách hàng", "giới thiệu sản phẩm"}

func loadCatalog(path string) ([]srcProduct, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var items []srcProduct
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("empty catalog %s", path)
	}
	return items, nil
}

func stableID(kind, key string) bson.ObjectID {
	hash := sha256.Sum256([]byte("aodora:sumire:" + kind + ":" + key))
	return bson.ObjectID(hash[:12])
}

func categoryFor(p srcProduct) categoryDef {
	has := func(tag string) bool {
		for _, t := range p.Tags {
			if strings.EqualFold(strings.TrimSpace(t), tag) {
				return true
			}
		}
		return false
	}
	title := strings.ToLower(p.Title)
	switch {
	case has("aodai4ta") || strings.Contains(title, "tà đôi"):
		return catTaDoi
	case has("AODAITAYCHEN"):
		return catTayChen
	case has("tet2026"):
		return catTet
	}
	return catDefault
}

// normalizeSize reduces "Size A", "Tú Trân / Size B" or "aa" to "A", "B", "AA".
func normalizeSize(raw string) string {
	value := strings.TrimSpace(raw)
	if i := strings.LastIndex(value, "/"); i >= 0 {
		value = strings.TrimSpace(value[i+1:])
	}
	value = strings.TrimSpace(sizePrefixRE.ReplaceAllString(value, ""))
	if strings.EqualFold(value, "default title") {
		return ""
	}
	return strings.ToUpper(value)
}

func parsePrice(raw string) (int64, error) {
	f, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil || f < 0 || math.IsNaN(f) || math.IsInf(f, 0) {
		return 0, fmt.Errorf("invalid price %q", raw)
	}
	return int64(math.Round(f)), nil
}

func htmlToText(src string) string {
	text := blockTagRE.ReplaceAllString(src, "\n")
	text = anyTagRE.ReplaceAllString(text, "")
	text = strings.ReplaceAll(html.UnescapeString(text), string(rune(0xA0)), " ")
	lines := []string{}
	for line := range strings.SplitSeq(text, "\n") {
		line = strings.Join(strings.Fields(line), " ")
		if line == "" || separatorRE.MatchString(line) {
			continue
		}
		lower := strings.ToLower(line)
		skip := false
		for _, prefix := range boilerplate {
			if strings.HasPrefix(lower, prefix) {
				skip = true
				break
			}
		}
		if !skip {
			lines = append(lines, line)
		}
	}
	return truncateLines(lines, maxDescription)
}

func truncateLines(lines []string, limit int) string {
	out, used := []string{}, 0
	for _, line := range lines {
		n := utf8.RuneCountInString(line)
		if used+n > limit {
			if len(out) == 0 {
				return string([]rune(line)[:limit-1]) + "…"
			}
			break
		}
		out = append(out, line)
		used += n + 1
	}
	return strings.Join(out, "\n")
}

func extractMaterial(text string) string {
	for _, re := range []*regexp.Regexp{materialRE, sectionMatRE} {
		if m := re.FindStringSubmatch(text); m != nil {
			return strings.TrimSpace(m[1])
		}
	}
	return ""
}

// colorRules are ordered: the first substring found in the lower-cased color name wins.
var colorRules = []struct{ sub, hex string }{
	{"tím huế", "#7B4B94"}, {"tím oải hương", "#A58FD1"}, {"tím cẩm", "#6A3D8F"}, {"tím nhạt", "#C9B3E6"}, {"tím", "#7E57C2"},
	{"xanh biển nhạt", "#8EC5E8"}, {"xanh biển", "#2F6FB5"}, {"xanh ngọc nhạt", "#9ADBD0"}, {"xanh ngọc", "#3CB8A4"},
	{"xanh cốm", "#B8D98B"}, {"xanh lông công", "#0F7F8F"}, {"xanh navy", "#1F2A56"}, {"xanh mint", "#A8E6CF"},
	{"bạc hà", "#A8E6CF"}, {"xanh thông", "#2E5E4E"}, {"xanh lá", "#3E8E41"}, {"xanh cổ vịt", "#2B7A78"},
	{"xanh đá", "#7FA3B8"}, {"xanh lơ", "#A9D6F5"}, {"xanh non", "#A5D65E"}, {"xanh", "#4C9F70"},
	{"hồng đậm", "#D81B60"}, {"hồng nhạt", "#F6C1CC"}, {"hồng ửng", "#F6C1CC"}, {"hồng", "#F06292"},
	{"đỏ", "#C62828"}, {"trắng kem", "#F5ECD7"}, {"kem", "#F5ECD7"}, {"trắng", "#F8F8F5"},
	{"vàng", "#F2C94C"}, {"cam", "#F28C28"}, {"nâu", "#7B5A3C"}, {"đen", "#1A1A1A"},
	{"đồng", "#B87333"}, {"xám", "#9E9E9E"}, {"bạc", "#C0C0C0"}, {"be", "#E8DCC8"},
}

// colorHex returns a swatch for a Vietnamese color name; ok is false when it fell back.
func colorHex(name string) (string, bool) {
	lower := strings.ToLower(strings.TrimSpace(name))
	lower = strings.TrimPrefix(lower, "nền ")
	// The base color is the first color word ("nền trắng hoa tím" is white); ties go to the more specific (earlier) rule.
	best, bestAt := "", -1
	for _, rule := range colorRules {
		if at := strings.Index(lower, rule.sub); at >= 0 && (bestAt < 0 || at < bestAt) {
			best, bestAt = rule.hex, at
		}
	}
	if bestAt < 0 {
		return fallbackHex, false
	}
	return best, true
}

func optionIndex(options []srcOption, names ...string) int {
	for i, o := range options {
		for _, n := range names {
			if strings.EqualFold(strings.TrimSpace(o.Name), n) {
				return i
			}
		}
	}
	return -1
}

func variantOption(v srcVariant, index int) string {
	var p *string
	switch index {
	case 0:
		p = v.Option1
	case 1:
		p = v.Option2
	case 2:
		p = v.Option3
	}
	if p == nil {
		return ""
	}
	return strings.TrimSpace(*p)
}

func imageURL(baseURL, handle, file string) string {
	return baseURL + "/" + imagePrefix + handle + "/" + file
}

// buildPlan converts source products into documents; it never touches the database or filesystem.
func buildPlan(items []srcProduct, baseURL string, defaultStock int64, now time.Time) (plan, error) {
	var out plan
	handles, slugs, skus := map[string]bool{}, map[string]bool{}, map[string]string{}
	usedCategories := map[string]bool{}
	for _, p := range items {
		if !handleRE.MatchString(p.Handle) {
			return out, fmt.Errorf("invalid handle %q", p.Handle)
		}
		if handles[p.Handle] {
			return out, fmt.Errorf("duplicate handle %q", p.Handle)
		}
		handles[p.Handle] = true
		if p.ProductType == skippedType {
			out.Skipped = append(out.Skipped, skipped{p.Handle, "product_type=" + skippedType})
			continue
		}
		doc, warnings, err := productDocument(p, baseURL, defaultStock, now)
		if err != nil {
			return out, fmt.Errorf("%s: %w", p.Handle, err)
		}
		slug := doc["slug"].(string)
		if slugs[slug] {
			return out, fmt.Errorf("duplicate slug %q", slug)
		}
		slugs[slug] = true
		for _, v := range doc["variants"].([]bson.M) {
			sku := v["sku"].(string)
			if owner, dup := skus[sku]; dup {
				// The source reuses SKUs across products; keep them unique with a stable handle-derived suffix.
				unique := sku + "-" + stableID("sku", p.Handle).Hex()[:6]
				warnings = append(warnings, fmt.Sprintf("%s: SKU %q already used by %s, renamed %q", p.Handle, sku, owner, unique))
				v["sku"], sku = unique, unique
				if skus[sku] != "" {
					return out, fmt.Errorf("%s: SKU %q still duplicated", p.Handle, sku)
				}
			}
			skus[sku] = p.Handle
		}
		out.Warnings = append(out.Warnings, warnings...)
		out.Products = append(out.Products, doc)
		usedCategories[categoryFor(p).Key] = true
		for _, im := range p.Images {
			out.Images = append(out.Images, imageRef{p.Handle, im.File})
		}
	}
	for i, c := range categories {
		if !usedCategories[c.Key] {
			continue
		}
		out.Categories = append(out.Categories, bson.M{
			"_id": stableID("category", c.Key), "name": c.Name, "slug": slugPrefix + c.Key,
			"description": "Danh mục áo dài Sumire Store dùng cho dữ liệu dev.", "parent_id": nil,
			"is_active": true, "sort_order": int64(100 + i), "version": int64(1), "created_at": now, "updated_at": now,
		})
	}
	return out, nil
}

func productDocument(p srcProduct, baseURL string, defaultStock int64, now time.Time) (bson.M, []string, error) {
	var warnings []string
	if strings.TrimSpace(p.Title) == "" {
		return nil, nil, fmt.Errorf("empty title")
	}
	if len(p.Images) == 0 {
		return nil, nil, fmt.Errorf("no images")
	}
	if len(p.Variants) == 0 {
		return nil, nil, fmt.Errorf("no variants")
	}
	sizeIdx := optionIndex(p.Options, "Kích thước", "Size")
	colorIdx := optionIndex(p.Options, "Màu sắc", "Color")

	images, urls := []bson.M{}, []string{}
	for i, im := range p.Images {
		if im.File == "" {
			return nil, nil, fmt.Errorf("image %d has no file", im.Order)
		}
		alt := p.Title
		if i > 0 {
			alt = fmt.Sprintf("%s — ảnh %d", p.Title, i+1)
		}
		src := imageURL(baseURL, p.Handle, im.File)
		images = append(images, bson.M{"src": src, "alt": alt})
		urls = append(urls, src)
	}

	variants := []bson.M{}
	sizes, colorNames := []string{}, []string{}
	seenSize, seenColor := map[string]bool{}, map[string]bool{}
	var price, compare int64 = -1, 0
	stock := int64(0)
	for i, v := range p.Variants {
		size, color := "", ""
		if sizeIdx >= 0 {
			size = normalizeSize(variantOption(v, sizeIdx))
		}
		if colorIdx >= 0 {
			color = variantOption(v, colorIdx)
		}
		vp, err := parsePrice(v.Price)
		if err != nil {
			return nil, nil, err
		}
		if price < 0 || vp < price {
			price = vp
		}
		if v.CompareAtPrice != nil {
			if cp, err := parsePrice(*v.CompareAtPrice); err == nil && cp > compare {
				compare = cp
			}
		}
		n := int64(0)
		if v.Available {
			n = defaultStock
		}
		sku := strings.TrimSpace(v.SKU)
		if sku == "" {
			sku = fmt.Sprintf("SUMIRE-%s-%d", strings.ToUpper(p.Handle), i+1)
		}
		stock += n
		variants = append(variants, bson.M{"sku": sku, "size": size, "color": color, "stock": n})
		if size != "" && !seenSize[size] {
			seenSize[size] = true
			sizes = append(sizes, size)
		}
		if color != "" && !seenColor[color] {
			seenColor[color] = true
			colorNames = append(colorNames, color)
		}
	}
	colors := []bson.M{}
	for _, name := range colorNames {
		hex, ok := colorHex(name)
		if !ok {
			warnings = append(warnings, fmt.Sprintf("%s: no hex for color %q (used %s)", p.Handle, name, hex))
		}
		colors = append(colors, bson.M{"name": name, "hex": hex})
	}

	description := htmlToText(p.BodyHTML)
	doc := bson.M{
		"_id": stableID("product", p.Handle), "category_id": stableID("category", categoryFor(p).Key),
		"name": strings.TrimSpace(p.Title), "slug": slugPrefix + p.Handle, "sku": productSKU(variants),
		"description": description, "collection": collectionName,
		"price": price, "compare_at_price": nil, "stock": stock, "variants": variants,
		"images": urls, "image": images[0], "gallery": images[1:], "colors": colors, "sizes": sizes,
		"is_active": true, "bespoke": false,
		"seed_source": seedSource, "source_url": p.SourceURL, "source_handle": p.Handle,
		"accessories": []bson.M{}, "trust": []bson.M{},
		"version": int64(1), "created_at": now, "updated_at": now,
	}
	if compare > price {
		doc["compare_at_price"], doc["old_price"] = compare, compare
	}
	if m := extractMaterial(description); m != "" {
		doc["material"] = m
	}
	if description != "" {
		doc["story"] = bson.M{"title": strings.TrimSpace(p.Title), "paragraphs": strings.Split(description, "\n"), "image": images[0]}
	}
	for _, t := range p.Tags {
		if strings.EqualFold(strings.TrimSpace(t), discontinuedTag) {
			doc["is_active"] = false
			doc["tag"] = bson.M{"label": "Ngừng sản xuất", "variant": "surface"}
			break
		}
	}
	return doc, warnings, nil
}

// productSKU strips the size suffix from the first variant SKU, e.g. AD000041AA/AA -> AD000041.
func productSKU(variants []bson.M) string {
	first := variants[0]["sku"].(string)
	if size := variants[0]["size"].(string); size != "" && strings.HasSuffix(first, size) && len(first) > len(size) {
		return strings.TrimSuffix(first, size)
	}
	return first
}
