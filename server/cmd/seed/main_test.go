package main

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestDemoCatalogReferencesStockAndGallery(t *testing.T) {
	data, err := os.ReadFile("../../seeds/forher.json")
	if err != nil {
		t.Fatal(err)
	}
	var seed manifest
	if err := json.Unmarshal(data, &seed); err != nil {
		t.Fatal(err)
	}
	baseURL := "https://images.example.test/aodora-images"
	categories, products := documents(seed, baseURL, time.Now())
	if len(categories) != 3 || len(products) != 3 {
		t.Fatalf("unexpected catalog size: %d categories, %d products", len(categories), len(products))
	}
	categoryIDs := map[any]bool{}
	for _, doc := range categories {
		categoryIDs[doc["_id"]] = true
	}
	skus := map[string]bool{}
	for _, doc := range products {
		if !categoryIDs[doc["category_id"]] {
			t.Fatalf("missing category for %s", doc["slug"])
		}
		stock := int64(0)
		for _, variant := range doc["variants"].([]bson.M) {
			sku := variant["sku"].(string)
			if skus[sku] {
				t.Fatalf("duplicate SKU %s", sku)
			}
			skus[sku] = true
			stock += variant["stock"].(int64)
		}
		if doc["stock"] != stock {
			t.Fatalf("aggregate stock differs from SKU stock: %v", doc)
		}
		urls := doc["images"].([]string)
		gallery := doc["gallery"].([]bson.M)
		if len(urls) != 3 || len(gallery) != 2 || doc["image"].(bson.M)["src"] != urls[0] {
			t.Fatalf("missing primary or gallery image: %v", doc)
		}
		for i, image := range gallery {
			if image["src"] != urls[i+1] || image["alt"] == "" || !strings.HasPrefix(urls[i+1], baseURL+"/demo/forher/") {
				t.Fatalf("invalid gallery image: %v", image)
			}
		}
		if doc["source_url"] == "" || doc["seed_source"] != seed.Source {
			t.Fatal("source provenance lost")
		}
		if _, err := bson.Marshal(doc); err != nil {
			t.Fatal(err)
		}
	}
	_, again := documents(seed, baseURL, time.Now().Add(time.Hour))
	for i := range products {
		if !reflect.DeepEqual(products[i]["_id"], again[i]["_id"]) {
			t.Fatal("rerunning would create duplicate product IDs")
		}
	}
}

func TestPublicBucketURLConfiguration(t *testing.T) {
	t.Setenv("RUSTFS_PUBLIC_URL", "")
	t.Setenv("RUSTFS_API_PORT", "9200")
	t.Setenv("RUSTFS_BUCKET", "test-images")
	if got := publicURL(); got != "http://localhost:9200/test-images" {
		t.Fatal(got)
	}
	t.Setenv("RUSTFS_PUBLIC_URL", "https://cdn.example.test/images/")
	if got := publicURL(); got != "https://cdn.example.test/images" {
		t.Fatal(got)
	}
	for _, invalid := range []string{"/relative", "ftp://host/images", "https://user:pass@host/images", "https://host/images?expires=1"} {
		if err := validate(manifest{}, invalid, t.TempDir()); err == nil {
			t.Fatalf("accepted invalid URL %s", invalid)
		}
	}
}
