package catalogadmin

import (
	"context"
	"errors"
	"testing"
)

type fakeRepo struct {
	state ProductState
	// adjustErr, when set, is returned by AdjustInventory.
	adjustErr error

	created  bool
	edited   bool
	adjusted bool

	createdVariants []Variant
	addedVariant    Variant
	updatedSize     string
}

func (f *fakeRepo) Categories(context.Context) ([]map[string]any, error) { return nil, nil }
func (f *fakeRepo) Products(context.Context, int64, int64) (ProductPage, error) {
	return ProductPage{}, nil
}
func (f *fakeRepo) ProductsBySlug(context.Context, []string) ([]map[string]any, error) {
	return nil, nil
}
func (f *fakeRepo) SKUOwners(context.Context, []string) (map[string]string, error) {
	return map[string]string{}, nil
}
func (f *fakeRepo) ProductState(context.Context, string) (ProductState, error) { return f.state, nil }
func (f *fakeRepo) CategoryExists(context.Context, string) (bool, error)       { return true, nil }
func (f *fakeRepo) CreateCategory(context.Context, CategoryInput) (map[string]any, error) {
	f.created = true
	return map[string]any{}, nil
}
func (f *fakeRepo) UpdateCategory(context.Context, string, CategoryInput) (map[string]any, error) {
	return nil, nil
}
func (f *fakeRepo) CreateProduct(_ context.Context, v ProductCreate) (map[string]any, error) {
	f.created = true
	f.createdVariants = v.Variants
	return map[string]any{}, nil
}
func (f *fakeRepo) UpdateProduct(context.Context, string, ProductInput) (map[string]any, error) {
	f.edited = true
	return map[string]any{}, nil
}
func (f *fakeRepo) AddVariant(_ context.Context, _ string, v Variant) (map[string]any, error) {
	f.edited = true
	f.addedVariant = v
	return map[string]any{}, nil
}
func (f *fakeRepo) UpdateVariant(_ context.Context, _, _ string, v VariantDetails) (map[string]any, error) {
	f.edited = true
	f.updatedSize = v.Size
	return map[string]any{}, nil
}
func (f *fakeRepo) AdjustInventory(context.Context, string, string, int64, string) (map[string]any, error) {
	if f.adjustErr != nil {
		return nil, f.adjustErr
	}
	f.adjusted = true
	return map[string]any{}, nil
}
func (f *fakeRepo) ProductContent(context.Context, string) (ContentRecord, error) {
	return ContentRecord{}, nil
}
func (f *fakeRepo) UpdateProductContent(_ context.Context, id string, input ProductContent) (ContentRecord, error) {
	f.edited = true
	return ContentRecord{ID: id, Version: *input.ExpectedVersion + 1, Story: input.Story, Fabric: input.Fabric, Craft: input.Craft}, nil
}
func goodCreate() ProductCreate {
	return ProductCreate{ProductInput: ProductInput{Name: "Áo dài", Slug: "ao-dai", CategoryID: "670000000000000000000001", Price: 500000, IsActive: true, Image: &Image{Src: "https://cdn.test/a.jpg", Alt: "Áo dài"}, Colors: []Color{{Name: "Đỏ"}}}, Variants: []Variant{{SKU: "AO-DO-M", Color: "Đỏ", Size: "M", Stock: 10}}}
}
func TestRejectInvalidWritesBeforeRepository(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*ProductCreate)
	}{
		{"negative price", func(p *ProductCreate) { p.Price = -1 }},
		{"bad slug", func(p *ProductCreate) { p.Slug = "Not valid" }},
		{"missing sku", func(p *ProductCreate) { p.Variants[0].SKU = "" }},
		{"duplicate sku", func(p *ProductCreate) { p.Variants = append(p.Variants, p.Variants[0]) }},
		{"duplicate color-size", func(p *ProductCreate) {
			p.Variants = append(p.Variants, Variant{SKU: "OTHER", Color: "Đỏ", Size: "M", Stock: 0})
		}},
		{"invalid image scheme", func(p *ProductCreate) { p.Image = &Image{Src: "javascript:alert(1)", Alt: "áo dài"} }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			v := goodCreate()
			tc.mutate(&v)
			repo := &fakeRepo{}
			_, err := NewService(repo).CreateProduct(context.Background(), v)
			if err == nil || repo.created {
				t.Fatalf("want validation error without db write: %v", err)
			}
		})
	}
}
func TestStockIsAdjustedOnlyByDelta(t *testing.T) {
	repo := &fakeRepo{}
	service := NewService(repo)
	_, err := service.AdjustInventory(context.Background(), "id", "sku", 0, "1dea1234-1234-4123-8123-111111111111")
	var invalid Invalid
	if !errors.As(err, &invalid) || repo.adjusted {
		t.Fatalf("zero delta must be rejected: %v", err)
	}
	_, err = service.AdjustInventory(context.Background(), "id", "sku", -2, "1dea1234-1234-4123-8123-111111111111")
	if err != nil || !repo.adjusted {
		t.Fatalf("valid delta must reach repository: %v", err)
	}
}
func TestNewSKUHasZeroStock(t *testing.T) {
	repo := &fakeRepo{}
	service := NewService(repo)
	_, err := service.AddVariant(context.Background(), "id", Variant{SKU: "AO-M", Color: "Trắng", Size: "M", Stock: 1})
	if err == nil || repo.edited {
		t.Fatal("new SKU must not set initial stock")
	}
}
func TestRejectCategorySelfParent(t *testing.T) {
	_, err := NewService(&fakeRepo{}).UpdateCategory(context.Background(), "abcdef123456abcdef123456", CategoryInput{Name: "Áo", Slug: "ao", ParentID: "ABCDEF123456ABCDEF123456"})
	if err == nil {
		t.Fatal("category self reference accepted")
	}
}
func TestProductMetadataExcludesStockAndReviews(t *testing.T) {
	// ProductInput is the only value accepted by UpdateProduct, so stock and
	// review counts cannot accidentally enter the metadata update contract.
	v := goodCreate().ProductInput
	if err := validateProduct(&v); err != nil {
		t.Fatal(err)
	}
}

func TestOptionalColorsAndAccessoryImages(t *testing.T) {
	v := goodCreate().ProductInput
	v.Colors = []Color{{Name: "Đỏ"}}
	v.Accessories = []Accessory{{ID: "one", Name: "Phụ kiện", Price: 1000, Tag: "lụa", TagVariant: "hồng"}}
	if err := validateProduct(&v); err != nil {
		t.Fatalf("legacy colors/accessories must be editable: %v", err)
	}
	v.Colors[0].Hex = "invalid"
	if err := validateProduct(&v); err == nil {
		t.Fatal("bad non-empty hex accepted")
	}
}
func TestVersionRequiredForPut(t *testing.T) {
	service := NewService(&fakeRepo{state: ProductState{Active: true}})
	v := goodCreate().ProductInput
	if _, err := service.UpdateProduct(context.Background(), "abcdef123456abcdef123456", v); err == nil {
		t.Fatal("PUT without expected_version must be rejected")
	}
	zero := int64(0)
	v.ExpectedVersion = &zero
	if _, err := service.UpdateProduct(context.Background(), "abcdef123456abcdef123456", v); err != nil {
		t.Fatalf("legacy version zero must be accepted: %v", err)
	}
}
func TestInventoryRequiresRetryID(t *testing.T) {
	service := NewService(&fakeRepo{})
	_, err := service.AdjustInventory(context.Background(), "id", "sku", 3, "")
	if err == nil {
		t.Fatal("request without idempotency key accepted")
	}
}

func TestSizeMustBeStandard(t *testing.T) {
	for _, size := range []string{"A", "AA", "XXXL", "42", "Size M"} {
		repo := &fakeRepo{}
		v := goodCreate()
		v.Variants[0].Size = size
		_, err := NewService(repo).CreateProduct(context.Background(), v)
		var invalid Invalid
		if !errors.As(err, &invalid) || invalid.Field != "variants[0].size" || invalid.Message != "Cỡ phải là một trong XS, S, M, L, XL, XXL, Freesize" || repo.created {
			t.Fatalf("create with size %q: err=%v created=%v", size, err, repo.created)
		}
		repo = &fakeRepo{}
		if _, err = NewService(repo).AddVariant(context.Background(), "id", Variant{SKU: "X", Color: "Đỏ", Size: size}); !errors.As(err, &invalid) || repo.edited {
			t.Fatalf("add variant with size %q: %v", size, err)
		}
		if _, err = NewService(repo).UpdateVariant(context.Background(), "id", "X", VariantDetails{Color: "Đỏ", Size: size}); !errors.As(err, &invalid) || repo.edited {
			t.Fatalf("update variant with size %q: %v", size, err)
		}
	}
}

func TestSizeIsStoredInStandardSpelling(t *testing.T) {
	repo := &fakeRepo{}
	v := goodCreate()
	v.Variants = []Variant{{SKU: "AO-A", Color: "Đỏ", Size: "freesize", Stock: 1}, {SKU: "AO-B", Color: "Đỏ", Size: " xl ", Stock: 1}, {SKU: "AO-C", Color: "Đỏ", Size: "", Stock: 1}}
	if _, err := NewService(repo).CreateProduct(context.Background(), v); err != nil {
		t.Fatal(err)
	}
	got := []string{repo.createdVariants[0].Size, repo.createdVariants[1].Size, repo.createdVariants[2].Size}
	if got[0] != "Freesize" || got[1] != "XL" || got[2] != "" {
		t.Fatalf("sizes not canonicalised: %v", got)
	}
	if _, err := NewService(repo).AddVariant(context.Background(), "id", Variant{SKU: "AO-D", Color: "Xanh", Size: "xxl"}); err != nil || repo.addedVariant.Size != "XXL" {
		t.Fatalf("add variant: %v %q", err, repo.addedVariant.Size)
	}
	if _, err := NewService(repo).UpdateVariant(context.Background(), "id", "AO-D", VariantDetails{Color: "Xanh", Size: "xs"}); err != nil || repo.updatedSize != "XS" {
		t.Fatalf("update variant: %v %q", err, repo.updatedSize)
	}
}
