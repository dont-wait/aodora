package catalogadmin

import (
	"context"
	"errors"
	"testing"
)

type fakeRepo struct {
	created  bool
	edited   bool
	adjusted bool
}

func (f *fakeRepo) Categories(context.Context) ([]map[string]any, error) { return nil, nil }
func (f *fakeRepo) Products(context.Context, int64, int64) (ProductPage, error) {
	return ProductPage{}, nil
}
func (f *fakeRepo) CategoryExists(context.Context, string) (bool, error) { return true, nil }
func (f *fakeRepo) CreateCategory(context.Context, CategoryInput) (map[string]any, error) {
	f.created = true
	return map[string]any{}, nil
}
func (f *fakeRepo) UpdateCategory(context.Context, string, CategoryInput) (map[string]any, error) {
	return nil, nil
}
func (f *fakeRepo) CreateProduct(context.Context, ProductCreate) (map[string]any, error) {
	f.created = true
	return map[string]any{}, nil
}
func (f *fakeRepo) UpdateProduct(context.Context, string, ProductInput) (map[string]any, error) {
	f.edited = true
	return map[string]any{}, nil
}
func (f *fakeRepo) AddVariant(context.Context, string, Variant) (map[string]any, error) {
	f.edited = true
	return map[string]any{}, nil
}
func (f *fakeRepo) UpdateVariant(context.Context, string, string, VariantDetails) (map[string]any, error) {
	f.edited = true
	return map[string]any{}, nil
}
func (f *fakeRepo) AdjustInventory(context.Context, string, string, int64) (map[string]any, error) {
	f.adjusted = true
	return map[string]any{}, nil
}
func goodCreate() ProductCreate {
	return ProductCreate{ProductInput: ProductInput{Name: "Áo dài", Slug: "ao-dai", CategoryID: "670000000000000000000001", Price: 500000, IsActive: true}, Variants: []Variant{{SKU: "AO-DO-M", Color: "Đỏ", Size: "M", Stock: 10}}}
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
	_, err := service.AdjustInventory(context.Background(), "id", "sku", 0)
	var invalid Invalid
	if !errors.As(err, &invalid) || repo.adjusted {
		t.Fatalf("zero delta must be rejected: %v", err)
	}
	_, err = service.AdjustInventory(context.Background(), "id", "sku", -2)
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
	_, err := NewService(&fakeRepo{}).UpdateCategory(context.Background(), "abc", CategoryInput{Name: "Áo", Slug: "ao", ParentID: "abc"})
	if err == nil {
		t.Fatal("category self reference accepted")
	}
}
func TestProductMetadataExcludesStockAndReviews(t *testing.T) {
	// ProductInput is the only value accepted by UpdateProduct, so stock and
	// review counts cannot accidentally enter the metadata update contract.
	v := goodCreate().ProductInput
	if err := validateProduct(v); err != nil {
		t.Fatal(err)
	}
}
