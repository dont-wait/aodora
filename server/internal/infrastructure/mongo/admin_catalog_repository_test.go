package mongo

import (
	app "github.com/your-org/aodora/server/internal/application/catalogadmin"
	"go.mongodb.org/mongo-driver/v2/bson"
	"testing"
)

func TestInventoryMutationGuardsVariantAndAggregateStock(t *testing.T) {
	id := bson.NewObjectID()
	filter, update := inventoryMutation(id, "SKU-M", -3, "11aa1234-1234-4123-8123-123456789012")
	if filter["_id"] != id {
		t.Fatal("wrong product")
	}
	if asMap(filter["stock"])["$gte"] != int64(3) {
		t.Fatal("aggregate stock is not guarded")
	}
	elem := asMap(asMap(filter["variants"])["$elemMatch"])
	if elem["sku"] != "SKU-M" || asMap(elem["stock"])["$gte"] != int64(3) {
		t.Fatalf("SKU stock guard missing: %v", elem)
	}
	inc := asMap(update["$inc"])
	if inc["stock"] != int64(-3) || inc["variants.$[target].stock"] != int64(-3) {
		t.Fatalf("not updating both stocks: %v", inc)
	}
}
func TestMetadataUpdateDoesNotIncludeInventoryOrRatings(t *testing.T) {
	set, _, err := productFields(app.ProductInput{Name: "Áo dài", Slug: "ao-dai", CategoryID: "670000000000000000000001", Price: 500000})
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"stock", "variants", "rating", "review_count", "story", "fabric", "craft"} {
		if _, ok := set[forbidden]; ok {
			t.Fatalf("metadata update overwrites %s", forbidden)
		}
	}
}

func TestOptimisticVersionFilterIncludesLegacyRecords(t *testing.T) {
	legacy := metadataVersionFilter(0)
	if _, ok := legacy["$or"]; !ok {
		t.Fatalf("missing legacy version condition: %v", legacy)
	}
	current := metadataVersionFilter(5)
	if current["version"] != int64(5) {
		t.Fatalf("wrong version: %v", current)
	}
}
func TestLegacyOldPriceIsClearedOnMetadataSave(t *testing.T) {
	input := app.ProductInput{CategoryID: "670000000000000000000001"}
	_, unset, err := productFields(input)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := unset["old_price"]; !ok {
		t.Fatal("stale old_price was not cleared")
	}
}
func TestInventoryMutationDeduplicatesInSameDocument(t *testing.T) {
	id := bson.NewObjectID()
	key := "33aa1234-1234-4123-8123-123456789012"
	filter, update := inventoryMutation(id, "AO-1", 3, key)
	if asMap(filter["inventory_operations.request_id"])["$ne"] != key {
		t.Fatal("missing idempotency guard")
	}
	if asMap(asMap(update["$push"])["inventory_operations"])["request_id"] != key {
		t.Fatal("missing operation log")
	}
}
func TestSizesAreDerivedFromSKUWithoutDuplicates(t *testing.T) {
	actual := sizesFromVariants([]app.Variant{{Size: "M"}, {Size: "L"}, {Size: "M"}})
	if len(actual) != 2 || actual[0] != "M" || actual[1] != "L" {
		t.Fatalf("sizes %v", actual)
	}
}

func TestInventoryArrayFiltersTargetSingleSKU(t *testing.T) {
	filters := inventoryArrayFilters("SKU-M", -2)
	if len(filters) != 1 {
		t.Fatalf("array filters: %v", filters)
	}
	f := asMap(filters[0])
	if f["target.sku"] != "SKU-M" || asMap(f["target.stock"])["$gte"] != int64(2) {
		t.Fatalf("wrong targeted SKU: %v", f)
	}
}
