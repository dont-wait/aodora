//go:build integration

package mongo

import (
	"context"
	"errors"
	"strings"
	"testing"

	app "github.com/your-org/aodora/server/internal/application/catalogadmin"
	"go.mongodb.org/mongo-driver/v2/bson"
	driver "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func indexModelUnique(key string) driver.IndexModel {
	return driver.IndexModel{Keys: bson.D{{Key: key, Value: 1}}, Options: options.Index().SetUnique(true)}
}

func fieldOfErr(err error) string {
	var invalid app.Invalid
	if errors.As(err, &invalid) {
		return invalid.Field
	}
	return ""
}

func TestDuplicateSlugAndSKUAreFieldErrorsOnMongo(t *testing.T) {
	db := scratchDatabase(t, "aodora_valid_test_")
	ctx := context.Background()
	// Unique indexes exist in the real schema; recreate the ones this test relies on.
	for _, spec := range []struct{ coll, key string }{{"products", "slug"}, {"products", "variants.sku"}, {"categories", "slug"}} {
		if _, err := db.Collection(spec.coll).Indexes().CreateOne(ctx, indexModelUnique(spec.key)); err != nil {
			t.Fatal(err)
		}
	}
	repo := NewAdminCatalogRepository(db)
	svc := app.NewService(repo, app.WithImagePrefix("http://localhost:9100/aodora-images"))
	img := &app.Image{Src: "http://localhost:9100/aodora-images/uploads/product/2026/03/a.jpg", Alt: "Áo"}

	cat, err := svc.CreateCategory(ctx, app.CategoryInput{Name: "Áo dài", Slug: "ao-dai", IsActive: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateCategory(ctx, app.CategoryInput{Name: "Áo dài 2", Slug: "ao-dai", IsActive: true}); fieldOfErr(err) != "slug" {
		t.Fatalf("duplicate category slug: %v", err)
	}
	catID := cat["id"].(string)
	newProduct := func(slug, sku string) app.ProductCreate {
		return app.ProductCreate{
			ProductInput: app.ProductInput{Name: "Áo " + slug, Slug: slug, CategoryID: catID, Price: 650000, IsActive: true, Image: img, Colors: []app.Color{{Name: "Đỏ"}}},
			Variants:     []app.Variant{{SKU: sku, Color: "đỏ", Size: "m", Stock: 5}},
		}
	}
	first, err := svc.CreateProduct(ctx, newProduct("ao-one", "AO-ONE-DO-M"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateProduct(ctx, newProduct("ao-one", "AO-OTHER-DO-M")); fieldOfErr(err) != "slug" {
		t.Fatalf("duplicate product slug: %v", err)
	}
	if _, err := svc.CreateProduct(ctx, newProduct("ao-two", "ao one do m")); fieldOfErr(err) != "variants[0].sku" {
		t.Fatalf("duplicate sku across products: %v", err)
	}
	id := first["id"].(string)
	if _, err := svc.AddVariant(ctx, id, app.Variant{SKU: "AO-ONE-DO-L", Color: "Xanh", Size: "L"}); fieldOfErr(err) != "color" {
		t.Fatalf("colour outside the list: %v", err)
	}
	if _, err := svc.AddVariant(ctx, id, app.Variant{SKU: "AO-ONE-DO-M", Color: "Đỏ", Size: "L"}); fieldOfErr(err) != "sku" {
		t.Fatalf("duplicate sku in product: %v", err)
	}
	if _, err := svc.AddVariant(ctx, id, app.Variant{SKU: "AO-ONE-DO-M2", Color: "Đỏ", Size: "M"}); fieldOfErr(err) != "color" {
		t.Fatalf("duplicate colour+size: %v", err)
	}
	if _, err := svc.AddVariant(ctx, id, app.Variant{SKU: "AO-ONE-DO-L", Color: "Đỏ", Size: "L"}); err != nil {
		t.Fatal(err)
	}
	state, err := repo.ProductState(ctx, id)
	if err != nil || state.Slug != "ao-one" || !state.Active || len(state.Variants) != 2 || len(state.Colors) != 1 || len(state.Images) != 1 {
		t.Fatalf("state %+v err=%v", state, err)
	}
	// Stock can never go negative and the message says how much is left.
	_, err = svc.AdjustInventory(ctx, id, "AO-ONE-DO-M", -6, "11111111-1111-4111-8111-111111111111")
	if fieldOfErr(err) != "delta" || !strings.Contains(err.Error(), "còn 5") {
		t.Fatalf("negative stock: %v", err)
	}
	if _, err := svc.AdjustInventory(ctx, id, "AO-ONE-DO-M", -5, "22222222-2222-4222-8222-222222222222"); err != nil {
		t.Fatal(err)
	}
	// Replaying the applied request must not be reported as insufficient stock.
	if _, err := svc.AdjustInventory(ctx, id, "AO-ONE-DO-M", -5, "22222222-2222-4222-8222-222222222222"); err != nil {
		t.Fatalf("idempotent replay: %v", err)
	}
	// Category cycles.
	child, err := svc.CreateCategory(ctx, app.CategoryInput{Name: "Con", Slug: "con", ParentID: catID, IsActive: true})
	if err != nil {
		t.Fatal(err)
	}
	version := int64(1)
	_, err = svc.UpdateCategory(ctx, catID, app.CategoryInput{Name: "Áo dài", Slug: "ao-dai", ParentID: child["id"].(string), IsActive: true, ExpectedVersion: &version})
	if fieldOfErr(err) != "parent_id" {
		t.Fatalf("category cycle: %v", err)
	}
	// A product cannot be turned on without a main image.
	hidden := newProduct("ao-hidden", "AO-HIDDEN-DO-M")
	hidden.IsActive = false
	hiddenOut, err := svc.CreateProduct(ctx, hidden)
	if err != nil {
		t.Fatal(err)
	}
	v := int64(1)
	upd := hidden.ProductInput
	upd.IsActive, upd.Image, upd.ExpectedVersion = true, nil, &v
	if _, err := svc.UpdateProduct(ctx, hiddenOut["id"].(string), upd); fieldOfErr(err) != "is_active" {
		t.Fatalf("activate without image: %v", err)
	}
}
