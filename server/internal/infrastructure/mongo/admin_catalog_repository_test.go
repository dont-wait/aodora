package mongo

import (
	app "github.com/your-org/aodora/server/internal/application/catalogadmin"
	"go.mongodb.org/mongo-driver/v2/bson"
	"testing"
)

func TestInventoryMutationGuardsVariantAndAggregateStock(t *testing.T) {
	id := bson.NewObjectID()
	filter, update := inventoryMutation(id, "SKU-M", -3)
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
	if inc["stock"] != int64(-3) || inc["variants.$.stock"] != int64(-3) {
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
