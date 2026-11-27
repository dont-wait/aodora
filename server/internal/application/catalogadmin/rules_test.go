package catalogadmin

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func validProduct() ProductInput {
	return ProductInput{
		Name: "Áo dài lụa", Slug: "ao-dai-lua", CategoryID: "670000000000000000000001", Price: 650000, IsActive: false,
		Image: &Image{Src: "https://cdn.test/a.jpg", Alt: "Áo dài lụa"},
	}
}

// fieldOf returns the Invalid.Field of err ("" when err is nil or not Invalid).
func fieldOf(err error) string {
	var invalid Invalid
	if errors.As(err, &invalid) {
		return invalid.Field
	}
	return ""
}

func TestProductRulesTable(t *testing.T) {
	long := func(n int) string { return strings.Repeat("a", n) }
	i64 := func(v int64) *int64 { return &v }
	cases := []struct {
		name   string
		mutate func(*ProductInput)
		field  string // "" = valid
	}{
		{"valid", func(p *ProductInput) {}, ""},
		{"name too short", func(p *ProductInput) { p.Name = "Áo" }, "name"},
		{"name only spaces", func(p *ProductInput) { p.Name = "     " }, "name"},
		{"name trimmed to 3", func(p *ProductInput) { p.Name = "  Áo   " }, "name"},
		{"name too long", func(p *ProductInput) { p.Name = long(151) }, "name"},
		{"name 150 ok", func(p *ProductInput) { p.Name = long(150) }, ""},
		{"slug uppercase", func(p *ProductInput) { p.Slug = "Ao-Dai" }, "slug"},
		{"slug double dash", func(p *ProductInput) { p.Slug = "ao--dai" }, "slug"},
		{"slug too long", func(p *ProductInput) { p.Slug = long(121) }, "slug"},
		{"slug 120 ok", func(p *ProductInput) { p.Slug = long(120) }, ""},
		{"category missing", func(p *ProductInput) { p.CategoryID = "" }, "category_id"},
		{"category not an id", func(p *ProductInput) { p.CategoryID = "abc" }, "category_id"},
		{"price below minimum", func(p *ProductInput) { p.Price = 999 }, "price"},
		{"price minimum", func(p *ProductInput) { p.Price = 1000 }, ""},
		{"price maximum", func(p *ProductInput) { p.Price = 1_000_000_000 }, ""},
		{"price above maximum", func(p *ProductInput) { p.Price = 1_000_000_001 }, "price"},
		{"compare equal to price", func(p *ProductInput) { p.CompareAtPrice = i64(650000) }, "compare_at_price"},
		{"compare below price", func(p *ProductInput) { p.CompareAtPrice = i64(1000) }, "compare_at_price"},
		{"compare above price", func(p *ProductInput) { p.CompareAtPrice = i64(700000) }, ""},
		{"description too long", func(p *ProductInput) { p.Description = long(5001) }, "description"},
		{"description 5000 ok", func(p *ProductInput) { p.Description = long(5000) }, ""},
		{"material too long", func(p *ProductInput) { p.Material = long(201) }, "material"},
		{"image alt empty", func(p *ProductInput) { p.Image.Alt = "  " }, "image.alt"},
		{"image alt too long", func(p *ProductInput) { p.Image.Alt = long(201) }, "image.alt"},
		{"image bad scheme", func(p *ProductInput) { p.Image.Src = "javascript:alert(1)" }, "image.src"},
		{"gallery 13 images", func(p *ProductInput) {
			for i := 0; i < 13; i++ {
				p.Gallery = append(p.Gallery, Image{Src: "https://cdn.test/g" + string(rune('a'+i)) + ".jpg", Alt: "x"})
			}
		}, "gallery"},
		{"gallery 12 images ok", func(p *ProductInput) {
			for i := 0; i < 12; i++ {
				p.Gallery = append(p.Gallery, Image{Src: "https://cdn.test/g" + string(rune('a'+i)) + ".jpg", Alt: "x"})
			}
		}, ""},
		{"gallery duplicate", func(p *ProductInput) {
			p.Gallery = []Image{{Src: "https://cdn.test/g.jpg", Alt: "x"}, {Src: "https://cdn.test/g.jpg", Alt: "y"}}
		}, "gallery[1].src"},
		{"gallery repeats main image", func(p *ProductInput) { p.Gallery = []Image{{Src: p.Image.Src, Alt: "x"}} }, "gallery[0].src"},
		{"gallery alt empty", func(p *ProductInput) { p.Gallery = []Image{{Src: "https://cdn.test/g.jpg", Alt: ""}} }, "gallery[0].alt"},
		{"colors 21", func(p *ProductInput) {
			for i := 0; i < 21; i++ {
				p.Colors = append(p.Colors, Color{Name: "Màu " + string(rune('a'+i))})
			}
		}, "colors"},
		{"color name empty", func(p *ProductInput) { p.Colors = []Color{{Name: " "}} }, "colors[0].name"},
		{"color name too long", func(p *ProductInput) { p.Colors = []Color{{Name: long(51)}} }, "colors[0].name"},
		{"color duplicate ignoring case and accents", func(p *ProductInput) { p.Colors = []Color{{Name: "Đỏ"}, {Name: "do"}} }, "colors[1].name"},
		{"color hex invalid", func(p *ProductInput) { p.Colors = []Color{{Name: "Đỏ"}, {Name: "Xanh", Hex: "#12"}} }, "colors[1].hex"},
		{"color hex ok", func(p *ProductInput) { p.Colors = []Color{{Name: "Đỏ", Hex: "#aabbcc"}} }, ""},
		{"accessories 21", func(p *ProductInput) {
			for i := 0; i < 21; i++ {
				p.Accessories = append(p.Accessories, Accessory{ID: "a" + string(rune('a'+i)), Name: "PK", Price: 0})
			}
		}, "accessories"},
		{"accessory name empty", func(p *ProductInput) { p.Accessories = []Accessory{{ID: "a1", Name: " "}} }, "accessories[0].name"},
		{"accessory name too long", func(p *ProductInput) { p.Accessories = []Accessory{{ID: "a1", Name: long(101)}} }, "accessories[0].name"},
		{"accessory description too long", func(p *ProductInput) { p.Accessories = []Accessory{{ID: "a1", Name: "PK", Description: long(501)}} }, "accessories[0].description"},
		{"accessory negative price", func(p *ProductInput) { p.Accessories = []Accessory{{ID: "a1", Name: "PK", Price: -1}} }, "accessories[0].price"},
		{"accessory price too high", func(p *ProductInput) { p.Accessories = []Accessory{{ID: "a1", Name: "PK", Price: 100_000_001}} }, "accessories[0].price"},
		{"accessory price free ok", func(p *ProductInput) { p.Accessories = []Accessory{{ID: "a1", Name: "PK", Price: 0}} }, ""},
		{"accessory duplicate id", func(p *ProductInput) { p.Accessories = []Accessory{{ID: "a1", Name: "PK"}, {ID: "a1", Name: "PK2"}} }, "accessories[1].id"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := validProduct()
			tc.mutate(&p)
			err := validateProduct(&p)
			if tc.field == "" {
				if err != nil {
					t.Fatalf("want valid, got %v", err)
				}
				return
			}
			if got := fieldOf(err); got != tc.field {
				t.Fatalf("field = %q (err %v), want %q", got, err, tc.field)
			}
		})
	}
}

func TestProductNormalisation(t *testing.T) {
	p := validProduct()
	p.Name = "  Áo dài  "
	p.Colors = []Color{{Name: " Đỏ đô ", Hex: "#aabbcc"}}
	if err := validateProduct(&p); err != nil {
		t.Fatal(err)
	}
	if p.Name != "Áo dài" || p.Colors[0].Name != "Đỏ đô" || p.Colors[0].Hex != "#AABBCC" {
		t.Fatalf("not normalised: %+v", p)
	}
}

func TestVariantRules(t *testing.T) {
	for _, tc := range []struct {
		name  string
		v     Variant
		field string
		sku   string
	}{
		{"lower-case with accents is normalised", Variant{SKU: " áo dài-đỏ m ", Color: "Đỏ", Size: "m", Stock: 3}, "", "AO-DAI-DO-M"},
		{"sku too short", Variant{SKU: "AB", Color: "Đỏ", Size: "M"}, "sku", ""},
		{"sku starts with dash", Variant{SKU: "-ABC", Color: "Đỏ", Size: "M"}, "sku", ""},
		{"sku bad character", Variant{SKU: "AB_C", Color: "Đỏ", Size: "M"}, "sku", ""},
		{"sku 40 ok", Variant{SKU: strings.Repeat("A", 40), Color: "Đỏ", Size: "M"}, "", strings.Repeat("A", 40)},
		{"sku 41", Variant{SKU: strings.Repeat("A", 41), Color: "Đỏ", Size: "M"}, "sku", ""},
		{"color empty", Variant{SKU: "ABC", Color: " ", Size: "M"}, "color", ""},
		{"size outside list", Variant{SKU: "ABC", Color: "Đỏ", Size: "A"}, "size", ""},
		{"stock negative", Variant{SKU: "ABC", Color: "Đỏ", Size: "M", Stock: -1}, "stock", ""},
		{"stock maximum", Variant{SKU: "ABC", Color: "Đỏ", Size: "M", Stock: 100000}, "", "ABC"},
		{"stock above maximum", Variant{SKU: "ABC", Color: "Đỏ", Size: "M", Stock: 100001}, "stock", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := tc.v
			err := validateVariant(&v)
			if got := fieldOf(err); got != tc.field {
				t.Fatalf("field = %q (%v), want %q", got, err, tc.field)
			}
			if tc.field == "" && v.SKU != tc.sku {
				t.Fatalf("sku = %q, want %q", v.SKU, tc.sku)
			}
		})
	}
}

func TestCategoryRules(t *testing.T) {
	long := func(n int) string { return strings.Repeat("a", n) }
	for _, tc := range []struct {
		name   string
		mutate func(*CategoryInput)
		field  string
	}{
		{"valid", func(c *CategoryInput) {}, ""},
		{"name too short", func(c *CategoryInput) { c.Name = "A" }, "name"},
		{"name too long", func(c *CategoryInput) { c.Name = long(101) }, "name"},
		{"slug invalid", func(c *CategoryInput) { c.Slug = "Áo dài" }, "slug"},
		{"sort negative", func(c *CategoryInput) { c.SortOrder = -1 }, "sort_order"},
		{"sort 9999 ok", func(c *CategoryInput) { c.SortOrder = 9999 }, ""},
		{"sort 10000", func(c *CategoryInput) { c.SortOrder = 10000 }, "sort_order"},
		{"description too long", func(c *CategoryInput) { c.Description = long(2001) }, "description"},
		{"image alt empty", func(c *CategoryInput) { c.Image = &Image{Src: "https://cdn.test/c.jpg"} }, "image.alt"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := CategoryInput{Name: "Áo dài", Slug: "ao-dai", IsActive: true}
			tc.mutate(&c)
			if got := fieldOf(validateCategory(&c)); got != tc.field {
				t.Fatalf("field = %q, want %q", got, tc.field)
			}
		})
	}
}

func goodWithVariants() ProductCreate {
	v := goodCreate()
	v.Variants = []Variant{{SKU: "AO-DO-M", Color: "đỏ", Size: "M", Stock: 5}, {SKU: "ao-do-l", Color: "Đỏ", Size: "L", Stock: 0}}
	return v
}

func TestCreateProductVariantErrorPaths(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*ProductCreate)
		field  string
	}{
		{"valid", func(p *ProductCreate) {}, ""},
		{"no colours", func(p *ProductCreate) { p.Colors = nil }, "colors"},
		{"colour not in list", func(p *ProductCreate) { p.Variants[1].Color = "Xanh" }, "variants[1].color"},
		{"duplicate sku", func(p *ProductCreate) { p.Variants[1].SKU = "ao do m" }, "variants[1].sku"},
		{"duplicate colour+size", func(p *ProductCreate) { p.Variants[1].Size = "M" }, "variants[1].color"},
		{"bad sku", func(p *ProductCreate) { p.Variants[1].SKU = "X" }, "variants[1].sku"},
		{"stock too high", func(p *ProductCreate) { p.Variants[0].Stock = 100001 }, "variants[0].stock"},
		{"active without image", func(p *ProductCreate) { p.Image = nil }, "is_active"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeRepo{}
			v := goodWithVariants()
			tc.mutate(&v)
			_, err := NewService(repo).CreateProduct(context.Background(), v)
			if got := fieldOf(err); got != tc.field {
				t.Fatalf("field = %q (%v), want %q", got, err, tc.field)
			}
			if tc.field == "" && !repo.created {
				t.Fatal("valid product not created")
			}
			if tc.field == "" {
				if repo.createdVariants[0].Color != "Đỏ" || repo.createdVariants[1].SKU != "AO-DO-L" {
					t.Fatalf("not normalised to the colour list / upper-case SKU: %+v", repo.createdVariants)
				}
			}
		})
	}
}

// dupRepo reports an existing slug / SKU owner.
type dupRepo struct {
	fakeRepo
	slugs  map[string]string // slug -> id
	owners map[string]string // sku (lower) -> product slug
}

func (d *dupRepo) ProductsBySlug(_ context.Context, slugs []string) ([]map[string]any, error) {
	var out []map[string]any
	for _, s := range slugs {
		if id, ok := d.slugs[s]; ok {
			out = append(out, map[string]any{"id": id, "slug": s})
		}
	}
	return out, nil
}
func (d *dupRepo) SKUOwners(context.Context, []string) (map[string]string, error) {
	return d.owners, nil
}

func TestDuplicateSlugAndSKUReportOnTheField(t *testing.T) {
	repo := &dupRepo{slugs: map[string]string{"ao-dai": "p1"}, owners: map[string]string{}}
	_, err := NewService(repo).CreateProduct(context.Background(), goodWithVariants())
	if fieldOf(err) != "slug" {
		t.Fatalf("duplicate slug: %v", err)
	}
	repo = &dupRepo{slugs: map[string]string{}, owners: map[string]string{"ao-do-l": "san-pham-khac"}}
	_, err = NewService(repo).CreateProduct(context.Background(), goodWithVariants())
	if fieldOf(err) != "variants[1].sku" || !strings.Contains(err.Error(), "san-pham-khac") {
		t.Fatalf("duplicate sku: %v", err)
	}
	// Updating a product may keep its own slug.
	repo = &dupRepo{slugs: map[string]string{"ao-dai": "abcdef123456abcdef123456"}, owners: map[string]string{}}
	repo.state = ProductState{ID: "abcdef123456abcdef123456", Active: true, Variants: []StateVariant{{SKU: "A-1", Color: "Đỏ", Size: "M"}}, Colors: []string{"Đỏ"}}
	zero := int64(0)
	v := goodCreate().ProductInput
	v.ExpectedVersion = &zero
	if _, err := NewService(repo).UpdateProduct(context.Background(), "abcdef123456abcdef123456", v); err != nil {
		t.Fatalf("own slug: %v", err)
	}
	repo.slugs["ao-dai"] = "ffffffffffffffffffffffff"
	if _, err := NewService(repo).UpdateProduct(context.Background(), "abcdef123456abcdef123456", v); fieldOf(err) != "slug" {
		t.Fatalf("slug of another product: %v", err)
	}
}

func TestConflictFromDatabaseIsAttributedNotA500(t *testing.T) {
	repo := &conflictRepo{dupRepo: dupRepo{slugs: map[string]string{}, owners: map[string]string{}}}
	_, err := NewService(repo).CreateProduct(context.Background(), goodWithVariants())
	// pre-checks pass, the insert races and returns ErrConflict; the service must not leak a generic error
	repo.slugs["ao-dai"] = "p9"
	if err != nil && !errors.Is(err, ErrConflict) && fieldOf(err) == "" {
		t.Fatalf("unexpected: %v", err)
	}
	repo2 := &conflictRepo{dupRepo: dupRepo{slugs: map[string]string{}, owners: map[string]string{}}, afterCreate: func(d *dupRepo) { d.slugs["ao-dai"] = "p9" }}
	_, err = NewService(repo2).CreateProduct(context.Background(), goodWithVariants())
	if fieldOf(err) != "slug" {
		t.Fatalf("slug race must surface as a slug error: %v", err)
	}
}

type conflictRepo struct {
	dupRepo
	afterCreate func(*dupRepo)
}

func (c *conflictRepo) CreateProduct(context.Context, ProductCreate) (map[string]any, error) {
	if c.afterCreate != nil {
		c.afterCreate(&c.dupRepo)
	}
	return nil, ErrConflict
}

func TestAdjustInventoryRules(t *testing.T) {
	id := "11111111-1111-4111-8111-111111111111"
	repo := &fakeRepo{}
	svc := NewService(repo)
	for _, delta := range []int64{0, 100001, -100001} {
		if _, err := svc.AdjustInventory(context.Background(), "p", "ABC", delta, id); fieldOf(err) != "delta" {
			t.Fatalf("delta %d: %v", delta, err)
		}
	}
	if _, err := svc.AdjustInventory(context.Background(), "p", "ABC", 100000, id); err != nil {
		t.Fatalf("100000 allowed: %v", err)
	}
	// Insufficient stock: the repository conflicts, the service says how much is left.
	repo = &fakeRepo{adjustErr: ErrConflict, state: ProductState{Variants: []StateVariant{{SKU: "ABC", Stock: 2}}}}
	_, err := NewService(repo).AdjustInventory(context.Background(), "p", "abc", -5, id)
	if fieldOf(err) != "delta" || !strings.Contains(err.Error(), "còn 2") {
		t.Fatalf("negative stock: %v", err)
	}
	// An idempotent retry of an already applied delta (stock now enough) keeps the original error.
	repo = &fakeRepo{adjustErr: ErrConflict, state: ProductState{Variants: []StateVariant{{SKU: "ABC", Stock: 9}}}}
	if _, err := NewService(repo).AdjustInventory(context.Background(), "p", "ABC", -5, id); !errors.Is(err, ErrConflict) {
		t.Fatalf("want ErrConflict, got %v", err)
	}
}

func TestColoursInUseCannotBeRemoved(t *testing.T) {
	repo := &fakeRepo{state: ProductState{ID: "abcdef123456abcdef123456", Active: true, Colors: []string{"Đỏ", "Xanh"}, Variants: []StateVariant{{SKU: "A-1", Color: "Xanh", Size: "M"}}}}
	zero := int64(0)
	v := goodCreate().ProductInput
	v.Colors = []Color{{Name: "Đỏ"}}
	v.ExpectedVersion = &zero
	_, err := NewService(repo).UpdateProduct(context.Background(), "abcdef123456abcdef123456", v)
	if fieldOf(err) != "colors" || !strings.Contains(err.Error(), "A-1") {
		t.Fatalf("%v", err)
	}
	v.Colors = []Color{{Name: "đỏ"}, {Name: "xanh"}}
	if _, err := NewService(repo).UpdateProduct(context.Background(), "abcdef123456abcdef123456", v); err != nil {
		t.Fatalf("renaming only the case keeps the colour: %v", err)
	}
}

func TestVariantColourMustBeInProductColours(t *testing.T) {
	repo := &fakeRepo{state: ProductState{Colors: []string{"Đỏ"}, Variants: []StateVariant{{SKU: "AO-DO-M", Color: "Đỏ", Size: "M"}}}}
	svc := NewService(repo)
	if _, err := svc.AddVariant(context.Background(), "p", Variant{SKU: "AO-XANH-M", Color: "Xanh", Size: "M"}); fieldOf(err) != "color" {
		t.Fatalf("colour outside list: %v", err)
	}
	if _, err := svc.AddVariant(context.Background(), "p", Variant{SKU: "ao-do-m", Color: "đỏ", Size: "L"}); fieldOf(err) != "sku" {
		t.Fatalf("duplicate sku (case-insensitive): %v", err)
	}
	if _, err := svc.AddVariant(context.Background(), "p", Variant{SKU: "AO-DO-M2", Color: "đỏ", Size: "m"}); fieldOf(err) != "color" {
		t.Fatalf("duplicate colour+size: %v", err)
	}
	if _, err := svc.AddVariant(context.Background(), "p", Variant{SKU: "AO-DO-L", Color: "đỏ", Size: "L"}); err != nil || repo.addedVariant.Color != "Đỏ" {
		t.Fatalf("valid add: %v %+v", err, repo.addedVariant)
	}
}

func TestActiveNeedsImageAndSKUWhenTurnedOn(t *testing.T) {
	zero := int64(0)
	v := goodCreate().ProductInput
	v.ExpectedVersion = &zero
	repo := &fakeRepo{state: ProductState{Active: false}}
	if _, err := NewService(repo).UpdateProduct(context.Background(), "abcdef123456abcdef123456", v); fieldOf(err) != "is_active" {
		t.Fatalf("no SKU: %v", err)
	}
	repo.state.Variants = []StateVariant{{SKU: "A-1"}}
	v.Image = nil
	if _, err := NewService(repo).UpdateProduct(context.Background(), "abcdef123456abcdef123456", v); fieldOf(err) != "is_active" {
		t.Fatalf("no image: %v", err)
	}
	v.IsActive = false
	if _, err := NewService(repo).UpdateProduct(context.Background(), "abcdef123456abcdef123456", v); err != nil {
		t.Fatalf("hidden product needs neither: %v", err)
	}
}

func TestOnlyOwnBucketImagesAreAcceptedForNewURLs(t *testing.T) {
	const bucket = "http://localhost:9100/aodora-images"
	own := bucket + "/uploads/product/2026/03/a.jpg"
	legacy := "https://cdn.legacy.test/old.jpg"
	zero := int64(0)
	repo := &fakeRepo{state: ProductState{ID: "abcdef123456abcdef123456", Active: true, Images: []string{legacy}, Variants: []StateVariant{{SKU: "A-1"}}}}
	svc := NewService(repo, WithImagePrefix(bucket+"/"))
	v := goodCreate().ProductInput
	v.ExpectedVersion = &zero
	v.Image = &Image{Src: own, Alt: "ok"}
	v.Gallery = []Image{{Src: legacy, Alt: "cũ"}}
	if _, err := svc.UpdateProduct(context.Background(), "abcdef123456abcdef123456", v); err != nil {
		t.Fatalf("bucket URL and an unchanged legacy URL must pass: %v", err)
	}
	v.Gallery = []Image{{Src: "https://evil.test/x.jpg", Alt: "ngoài"}}
	if _, err := svc.UpdateProduct(context.Background(), "abcdef123456abcdef123456", v); fieldOf(err) != "gallery[0].src" {
		t.Fatalf("external gallery URL: %v", err)
	}
	v.Gallery = nil
	v.Accessories = []Accessory{{ID: "a1", Name: "PK", Image: &Image{Src: "https://evil.test/a.jpg", Alt: "pk"}}}
	if _, err := svc.UpdateProduct(context.Background(), "abcdef123456abcdef123456", v); fieldOf(err) != "accessories[0].image.src" {
		t.Fatalf("external accessory URL: %v", err)
	}
	// A prefix that merely starts with the bucket text is not the bucket.
	v.Accessories = nil
	v.Image = &Image{Src: bucket + "-evil/x.jpg", Alt: "x"}
	if _, err := svc.UpdateProduct(context.Background(), "abcdef123456abcdef123456", v); fieldOf(err) != "image.src" {
		t.Fatalf("look-alike bucket: %v", err)
	}
	// Creating never reuses legacy URLs.
	c := goodCreate()
	c.Image = &Image{Src: legacy, Alt: "x"}
	if _, err := svc.CreateProduct(context.Background(), c); fieldOf(err) != "image.src" {
		t.Fatalf("create with external image: %v", err)
	}
	if _, err := svc.CreateCategory(context.Background(), CategoryInput{Name: "Áo dài", Slug: "ao-dai", Image: &Image{Src: legacy, Alt: "x"}}); fieldOf(err) != "image.src" {
		t.Fatalf("category external image: %v", err)
	}
}

// catRepo serves a fixed category list.
type catRepo struct {
	fakeRepo
	cats []map[string]any
}

func (c *catRepo) Categories(context.Context) ([]map[string]any, error) { return c.cats, nil }

func TestCategoryParentCycleAndDuplicateSlug(t *testing.T) {
	a, b, c := "aaaaaaaaaaaaaaaaaaaaaaaa", "bbbbbbbbbbbbbbbbbbbbbbbb", "cccccccccccccccccccccccc"
	repo := &catRepo{cats: []map[string]any{
		{"id": a, "slug": "a", "name": "Aa", "parent_id": ""},
		{"id": b, "slug": "b", "name": "Bb", "parent_id": a},
		{"id": c, "slug": "c", "name": "Cc", "parent_id": b},
	}}
	svc := NewService(repo)
	zero := int64(0)
	upd := func(id string, in CategoryInput) error {
		in.ExpectedVersion = &zero
		_, err := svc.UpdateCategory(context.Background(), id, in)
		return err
	}
	if err := upd(a, CategoryInput{Name: "Aa", Slug: "a", ParentID: c}); fieldOf(err) != "parent_id" {
		t.Fatalf("grandchild as parent: %v", err)
	}
	if err := upd(a, CategoryInput{Name: "Aa", Slug: "a", ParentID: a}); fieldOf(err) != "parent_id" {
		t.Fatalf("self as parent: %v", err)
	}
	if err := upd(c, CategoryInput{Name: "Cc", Slug: "c", ParentID: a}); err != nil {
		t.Fatalf("moving up is fine: %v", err)
	}
	if err := upd(c, CategoryInput{Name: "Cc", Slug: "b"}); fieldOf(err) != "slug" {
		t.Fatalf("slug of another category: %v", err)
	}
	if _, err := svc.CreateCategory(context.Background(), CategoryInput{Name: "Mới", Slug: "a"}); fieldOf(err) != "slug" {
		t.Fatalf("create duplicate slug: %v", err)
	}
}
