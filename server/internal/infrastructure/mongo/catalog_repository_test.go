package mongo

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	model "github.com/your-org/aodora/server/internal/domain/catalog"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestProductFilterCombinesColorAndSizeInOneSKU(t *testing.T) {
	low, high := int64(100), int64(300)
	query := productFilter(model.Filter{
		Search: "[lụa]", Color: "Đỏ", Size: "M", Material: "lụa", MinPrice: &low, MaxPrice: &high,
	}, []string{"680000000000000000000001"})
	data, err := bson.MarshalExtJSON(query, false, false)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, word := range []string{"$elemMatch", "$in", "$gte", "$lte", "is_active", "description", "material"} {
		if !strings.Contains(text, word) {
			t.Fatalf("missing %s: %s", word, text)
		}
	}
	if !strings.Contains(text, `\\[lụa\\]`) {
		t.Fatalf("search regex was not escaped: %s", text)
	}
	// Check the concrete nested predicate rather than independent variants.color/size.
	terms := query["$and"].(bson.A)
	combined := terms[len(terms)-1].(bson.M)["$or"].(bson.A)
	variant := combined[0].(bson.M)["variants"].(bson.M)["$elemMatch"].(bson.M)
	if len(variant) != 2 || variant["size"].(bson.Regex).Pattern != "^M$" {
		t.Fatalf("not same-SKU filter: %#v", variant)
	}
}

func TestStableSortTieBreakers(t *testing.T) {
	for _, sortKey := range []string{"newest", "price-asc", "price-desc", "name_asc", "name_desc"} {
		got := productSort(sortKey)
		if len(got) != 2 || got[1].Key != "_id" {
			t.Fatalf("unstable sort for %s: %v", sortKey, got)
		}
	}
}

func TestProductFromBSONAllowsDecimalAndEmbeddedContent(t *testing.T) {
	id, _ := bson.ObjectIDFromHex("680000000000000000000001")
	cat, _ := bson.ObjectIDFromHex("670000000000000000000001")
	decimal, _ := bson.ParseDecimal128("2190000.5")
	raw := bson.M{
		"_id": id, "category_id": cat, "slug": "ao-b", "name": "Áo B", "price": decimal,
		"images": bson.A{"https://example.com/image.jpg"}, "stock": int32(3),
		"variants": bson.A{bson.M{"sku": "B-RED-S", "size": "S", "color": "Đỏ", "stock": int32(0)},
			bson.M{"sku": "B-RED-M", "size": "M", "color": "Đỏ", "stock": int32(3)}},
		"story":                 bson.M{"title": "B", "paragraphs": bson.A{"Một", "Hai"}},
		"fabric":                bson.M{"care": bson.A{bson.M{"title": "Giặt", "description": "Bằng tay"}}},
		"private_customer_data": "never exposed",
	}
	product, err := productFromBSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if product.Price != json.Number("2190000.5") || product.Image.Alt != "Áo B" || product.Story.Title != "B" {
		t.Fatalf("failed decode %+v", product)
	}
	if !reflect.DeepEqual(product.Sizes, []string{"M", "S"}) {
		t.Fatalf("size variants %#v", product.Sizes)
	}
	product.SetAvailability()
	if !product.ReadyAvailable || product.Variants[0].Available || !product.Variants[1].Available {
		t.Fatalf("SKU availability failed %+v", product.Variants)
	}
	data, err := json.Marshal(product)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "private_customer_data") {
		t.Fatal("leaked private data")
	}
}
