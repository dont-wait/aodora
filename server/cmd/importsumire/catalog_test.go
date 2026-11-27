package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func ptr(s string) *string { return &s }

func sample() srcProduct {
	return srcProduct{
		Handle: "ao-dai-van-hoa-do", Title: "Áo dài tà đôi Vạn Hoa - Đỏ", ProductType: "Áo dài",
		Tags:     []string{"addangom", "tet2026"},
		BodyHTML: "<p>Tên sản phẩm:&nbsp;Áo dài</p><p>_______</p><p>Chị bạn khách hàng vui lòng tham khảo.</p><p>Chất liệu: tơ tencel óng ả, mềm mại.</p>",
		Options:  []srcOption{{Name: "Màu sắc", Position: 1}, {Name: "Kích thước", Position: 2}},
		Variants: []srcVariant{
			{SKU: "AD000041AA", Option1: ptr("Đỏ"), Option2: ptr("AA"), Price: "1380000.00", CompareAtPrice: ptr("1500000"), Available: true},
			{SKU: "AD000041A", Option1: ptr("Đỏ"), Option2: ptr("Size A"), Price: "1380000", Available: false},
		},
		Images: []srcImage{{Order: 1, File: "01.png"}, {Order: 2, File: "02.jpg"}},
	}
}

func TestNormalizeSize(t *testing.T) {
	for in, want := range map[string]string{"Size A": "S", "A": "S", "aa": "XS", "Tú Trân / Size B": "M", "Default Title": "", " Size  D ": "XL", "Size C": "L", "xxl": "XXL", "freesize": "Freesize", "Size XL": "XL"} {
		if got := normalizeSize(in); got != want {
			t.Errorf("normalizeSize(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestHTMLToTextDropsBoilerplate(t *testing.T) {
	got := htmlToText(sample().BodyHTML)
	if strings.Contains(got, "Tên sản phẩm") || strings.Contains(got, "___") || strings.Contains(got, "Chị bạn") || strings.Contains(got, "&nbsp;") {
		t.Fatalf("boilerplate left in %q", got)
	}
	if !strings.Contains(got, "tencel") {
		t.Fatalf("lost content: %q", got)
	}
	if m := extractMaterial(got); m != "tơ tencel óng ả" {
		t.Fatalf("material = %q", m)
	}
	if m := extractMaterial("Mô tả\n[Chất liệu]\nTơ tencel ánh may 02 lớp: lớp ngoài hoa\n[Thiết kế]"); m != "Tơ tencel ánh may 02 lớp" {
		t.Fatalf("section material = %q", m)
	}
	if m := extractMaterial("Rải khắp thân áo, chất liệu tơ tencel óng ả càng làm tôn lên vẻ rộn ràng"); m != "" {
		t.Fatalf("prose must not be mistaken for a material label: %q", m)
	}
	if m := extractMaterial("Chất liệu: Gấm 2 da\nCỡ: S"); m != "Gấm 2 da" {
		t.Fatalf("labelled material = %q", m)
	}
}

func TestHTMLToTextTruncates(t *testing.T) {
	long := strings.Repeat("<p>"+strings.Repeat("a", 400)+"</p>", 5)
	got := htmlToText(long)
	if n := len([]rune(got)); n > maxDescription || n < 400 || !strings.HasPrefix(got, strings.Repeat("a", 400)) {
		t.Fatalf("multiline description must keep leading lines within the limit, got %d runes", n)
	}
	if single := htmlToText("<p>" + strings.Repeat("b", 5000) + "</p>"); len([]rune(single)) != maxDescription {
		t.Fatalf("single long line must be cut to %d runes, got %d", maxDescription, len([]rune(single)))
	}
}

func TestCategoryFor(t *testing.T) {
	cases := []struct {
		title string
		tags  []string
		want  categoryDef
	}{
		{"Áo dài tà đôi X", nil, catTaDoi},
		{"Y", []string{"aodai4ta"}, catTaDoi},
		{"Y", []string{"AODAITAYCHEN"}, catTayChen},
		{"Y", []string{"addangom", "tet2026"}, catTet},
		{"Y", []string{"addangom"}, catDefault},
	}
	for _, c := range cases {
		if got := categoryFor(srcProduct{Title: c.title, Tags: c.tags}); got != c.want {
			t.Errorf("%q %v -> %v, want %v", c.title, c.tags, got, c.want)
		}
	}
}

func TestColorHex(t *testing.T) {
	for name, want := range map[string]string{"Đỏ": "#C62828", "Nền trắng hoa tím": "#F8F8F5", "Xanh biển nhạt": "#8EC5E8", "Tím oải hương": "#A58FD1", "Hồng viền xanh cốm": "#F06292", "Nâu hoa xanh": "#7B5A3C"} {
		if got, ok := colorHex(name); !ok || got != want {
			t.Errorf("colorHex(%q) = %s %v, want %s", name, got, ok, want)
		}
	}
	if hex, ok := colorHex("Nguyệt Mai"); ok || hex != fallbackHex {
		t.Errorf("unknown color must fall back, got %s %v", hex, ok)
	}
}

func TestBuildPlanProductDocument(t *testing.T) {
	now := time.Now()
	p, err := buildPlan([]srcProduct{sample()}, "http://x/bucket", 5, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Products) != 1 || len(p.Categories) != 1 || len(p.Images) != 2 {
		t.Fatalf("unexpected plan: %+v", p)
	}
	doc := p.Products[0]
	if doc["slug"] != "sumire-ao-dai-van-hoa-do" || doc["sku"] != "AD000041" || doc["price"] != int64(1380000) {
		t.Fatalf("bad identity fields: %v %v %v", doc["slug"], doc["sku"], doc["price"])
	}
	if doc["compare_at_price"] != int64(1500000) || doc["old_price"] != int64(1500000) {
		t.Fatalf("compare price missing: %v", doc)
	}
	if doc["stock"] != int64(5) || doc["is_active"] != true {
		t.Fatalf("stock/active wrong: %v %v", doc["stock"], doc["is_active"])
	}
	variants := doc["variants"].([]bson.M)
	if variants[0]["size"] != "XS" || variants[1]["size"] != "S" || variants[1]["stock"] != int64(0) || variants[0]["color"] != "Đỏ" {
		t.Fatalf("bad variants: %v", variants)
	}
	if got := doc["sizes"].([]string); len(got) != 2 || got[0] != "XS" || got[1] != "S" {
		t.Fatalf("bad sizes: %v", got)
	}
	urls := doc["images"].([]string)
	if urls[0] != "http://x/bucket/catalog/sumire/ao-dai-van-hoa-do/01.png" || urls[1] != "http://x/bucket/catalog/sumire/ao-dai-van-hoa-do/02.jpg" {
		t.Fatalf("bad urls: %v", urls)
	}
	if doc["image"].(bson.M)["src"] != urls[0] || len(doc["gallery"].([]bson.M)) != 1 {
		t.Fatalf("bad image/gallery")
	}
	if doc["category_id"] != p.Categories[0]["_id"] {
		t.Fatal("category reference broken")
	}
	if doc["material"] != "tơ tencel óng ả" || doc["seed_source"] != seedSource {
		t.Fatalf("material/source wrong: %v %v", doc["material"], doc["seed_source"])
	}
}

func TestDiscontinuedIsInactiveAndTagged(t *testing.T) {
	s := sample()
	s.Tags = append(s.Tags, "vd-ngung-sx")
	p, err := buildPlan([]srcProduct{s}, "http://x/b", 5, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if p.Products[0]["is_active"] != false || p.Products[0]["tag"].(bson.M)["label"] != "Ngừng sản xuất" {
		t.Fatalf("discontinued product not hidden: %v", p.Products[0])
	}
}

func TestOtherTypeSkippedAndNoSizeOption(t *testing.T) {
	other := sample()
	other.Handle, other.ProductType = "phu-kien", skippedType
	plain := sample()
	plain.Handle, plain.Options = "khong-size", []srcOption{{Name: "Tiêu đề"}}
	plain.Variants = []srcVariant{{SKU: "", Option1: ptr("Default Title"), Price: "319000", Available: true}}
	p, err := buildPlan([]srcProduct{other, plain}, "http://x/b", 3, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Skipped) != 1 || len(p.Products) != 1 {
		t.Fatalf("skip logic wrong: %+v", p.Skipped)
	}
	v := p.Products[0]["variants"].([]bson.M)[0]
	if v["sku"] != "SUMIRE-KHONG-SIZE-1" || v["size"] != "" || v["stock"] != int64(3) {
		t.Fatalf("bad no-size variant: %v", v)
	}
	if len(p.Products[0]["sizes"].([]string)) != 0 {
		t.Fatal("sizes must be empty when product has no size option")
	}
}

func TestDuplicateSKUIsRenamedStably(t *testing.T) {
	a, b := sample(), sample()
	b.Handle = "ao-dai-khac"
	p, err := buildPlan([]srcProduct{a, b}, "http://x/b", 5, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	first := p.Products[0]["variants"].([]bson.M)[0]["sku"]
	second := p.Products[1]["variants"].([]bson.M)[0]["sku"].(string)
	if first != "AD000041AA" || second == first || !strings.HasPrefix(second, "AD000041AA-") || len(p.Warnings) == 0 {
		t.Fatalf("sku collision not handled: %v %v %v", first, second, p.Warnings)
	}
	seen := map[string]bool{}
	for _, doc := range p.Products {
		for _, v := range doc["variants"].([]bson.M) {
			sku := v["sku"].(string)
			if seen[sku] {
				t.Fatalf("duplicate variant SKU left in plan: %s", sku)
			}
			seen[sku] = true
		}
	}
	if p.Products[0]["sku"] == p.Products[1]["sku"] {
		t.Fatalf("product-level SKU still duplicated: %v", p.Products[1]["sku"])
	}
	again, _ := buildPlan([]srcProduct{a, b}, "http://x/b", 5, time.Now())
	for i := range p.Products {
		got, want := again.Products[i]["variants"].([]bson.M), p.Products[i]["variants"].([]bson.M)
		for j := range want {
			if got[j]["sku"] != want[j]["sku"] {
				t.Fatalf("renamed SKU %v is not stable across runs (%v)", want[j]["sku"], got[j]["sku"])
			}
		}
	}
}

func TestIDsAreStable(t *testing.T) {
	a1, a2 := stableID("product", "a"), stableID("product", "a")
	if a1 != a2 || a1 == stableID("product", "b") || a1 == stableID("category", "a") {
		t.Fatal("stableID must be deterministic and namespaced")
	}
}

func TestBuildPlanRejectsBadInput(t *testing.T) {
	bad := sample()
	bad.Handle = "../evil"
	if _, err := buildPlan([]srcProduct{bad}, "http://x/b", 5, time.Now()); err == nil {
		t.Fatal("unsafe handle accepted")
	}
	dup := sample()
	if _, err := buildPlan([]srcProduct{dup, dup}, "http://x/b", 5, time.Now()); err == nil {
		t.Fatal("duplicate handle accepted")
	}
	noImg := sample()
	noImg.Images = nil
	if _, err := buildPlan([]srcProduct{noImg}, "http://x/b", 5, time.Now()); err == nil {
		t.Fatal("product without images accepted")
	}
}

func TestValidateImagesChecksMagicBytes(t *testing.T) {
	dir := t.TempDir()
	write := func(name string, data []byte) {
		path := filepath.Join(dir, "h", name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	png := append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, 8)...)
	jpg := append([]byte{0xFF, 0xD8, 0xFF, 0xE0}, make([]byte, 12)...)
	webp := append([]byte("RIFF\x00\x00\x00\x00WEBP"), make([]byte, 4)...)
	write("a.png", png)
	write("b.jpg", jpg)
	write("c.webp", webp)
	ok := []imageRef{{"h", "a.png"}, {"h", "b.jpg"}, {"h", "c.webp"}}
	if err := validateImages(ok, dir); err != nil {
		t.Fatal(err)
	}
	write("d.png", jpg) // wrong content for the extension
	for _, bad := range []imageRef{{"h", "d.png"}, {"h", "missing.png"}, {"h", "../x.png"}} {
		if err := validateImages([]imageRef{bad}, dir); err == nil {
			t.Errorf("%v accepted", bad)
		}
	}
	write("e.png", []byte("short"))
	if err := validateImages([]imageRef{{"h", "e.png"}}, dir); err == nil {
		t.Error("truncated file accepted")
	}
}

func TestIsLocalMongo(t *testing.T) {
	for uri, want := range map[string]bool{
		"mongodb://localhost:27017/?authSource=admin":         true,
		"mongodb://devuser@localhost:27017/?authSource=admin": true,
		"mongodb://127.0.0.1:27017":                           true,
		"mongodb://[::1]:27017/db":                            true,
		"mongodb://devuser@db.prod.example.com:27017/":        false,
		"mongodb://localhost:27017,remote.example.com:27017/": false,
		"mongodb+srv://cluster0.abc.mongodb.net/":             false,
		"": false,
	} {
		if got := isLocalMongo(uri); got != want {
			t.Errorf("isLocalMongo(%q) = %v, want %v", uri, got, want)
		}
	}
}

func TestImagesSortedByOrderAndNoStory(t *testing.T) {
	s := sample()
	s.Images = []srcImage{{Order: 2, File: "02.jpg"}, {Order: 1, File: "01.png"}}
	p, err := buildPlan([]srcProduct{s}, "http://x/b", 5, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	doc := p.Products[0]
	if got := doc["image"].(bson.M)["src"]; got != "http://x/b/catalog/sumire/ao-dai-van-hoa-do/01.png" {
		t.Fatalf("cover image must be order 1, got %v", got)
	}
	if _, has := doc["story"]; has {
		t.Fatal("description must not be duplicated into the story tab")
	}
}

func TestOptionsUsePositionNotArrayIndex(t *testing.T) {
	s := sample()
	// Listed in reverse array order; Position says size is option1.
	s.Options = []srcOption{{Name: "Màu sắc", Position: 2}, {Name: "Kích thước", Position: 1}}
	s.Variants = []srcVariant{{SKU: "X1A", Option1: ptr("Size A"), Option2: ptr("Đỏ"), Price: "100000", Available: true}}
	p, err := buildPlan([]srcProduct{s}, "http://x/b", 5, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	v := p.Products[0]["variants"].([]bson.M)[0]
	if v["size"] != "S" || v["color"] != "Đỏ" {
		t.Fatalf("options resolved by array index instead of position: %v", v)
	}
}

func TestVariantPricesKeptAndWarned(t *testing.T) {
	s := sample()
	s.Variants[0].Price, s.Variants[1].Price = "680000", "880000"
	s.Variants[0].CompareAtPrice = nil
	p, err := buildPlan([]srcProduct{s}, "http://x/b", 5, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	vs := p.Products[0]["variants"].([]bson.M)
	if p.Products[0]["price"] != int64(680000) || vs[0]["price"] != int64(680000) || vs[1]["price"] != int64(880000) {
		t.Fatalf("variant prices lost: %v", vs)
	}
	found := false
	for _, w := range p.Warnings {
		found = found || strings.Contains(w, "different prices")
	}
	if !found {
		t.Fatalf("missing different-price warning: %v", p.Warnings)
	}
}

func TestJPEGExtensionIsRejected(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "h", "01.jpeg")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append([]byte{0xFF, 0xD8, 0xFF, 0xE0}, make([]byte, 12)...), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := validateImages([]imageRef{{"h", "01.jpeg"}}, dir); err == nil {
		t.Fatal(".jpeg is not synced by rustfs-sumire and must be rejected")
	}
}

func TestSortSizesFollowsStandardOrder(t *testing.T) {
	got := sortSizes([]string{"XL", "XS", "Custom", "M", "S"})
	want := []string{"XS", "S", "M", "XL", "Custom"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("sortSizes = %v, want %v", got, want)
		}
	}
}
