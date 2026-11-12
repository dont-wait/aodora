package mongo

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

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
	for _, word := range []string{"$elemMatch", "$in", "$gte", "$lte", "is_active", "material"} {
		if !strings.Contains(text, word) {
			t.Fatalf("missing %s: %s", word, text)
		}
	}
	if !strings.Contains(text, `"$text"`) {
		t.Fatalf("search must use MongoDB text index: %s", text)
	}
	foundSearch := false
	for _, term := range query["$and"].(bson.A) {
		if value, ok := term.(bson.M)["$text"]; ok {
			foundSearch = value.(bson.M)["$search"] == "[lụa]"
		}
	}
	if !foundSearch {
		t.Fatalf("wrong search term: %s", text)
	}
	// Check the concrete nested predicate rather than independent variants.color/size.
	terms := query["$and"].(bson.A)
	combined := terms[len(terms)-1].(bson.M)["$or"].(bson.A)
	variant := combined[0].(bson.M)["variants"].(bson.M)["$elemMatch"].(bson.M)
	if len(variant) != 2 || variant["size"].(bson.Regex).Pattern != "^M$" {
		t.Fatalf("not same-SKU filter: %#v", variant)
	}
	color, ok := variant["color"].(bson.Regex)
	if !ok || color.Pattern != "^Đỏ$" || color.Options != "i" {
		t.Fatalf("wrong color constraint: %#v", variant["color"])
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

func TestRejectNonFiniteDecimalPrices(t *testing.T) {
	for _, value := range []string{"Infinity", "-Infinity", "NaN"} {
		t.Run(value, func(t *testing.T) {
			n, err := bson.ParseDecimal128(value)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := asNumber(n); err == nil {
				t.Fatalf("non-finite decimal %s was accepted", value)
			}
			_, err = productFromBSON(bson.M{"price": n})
			if err == nil {
				t.Fatalf("product with non-finite price %s accepted", value)
			}
			_, err = productFromBSON(bson.M{"price": int64(1), "old_price": n})
			if err == nil {
				t.Fatalf("product with non-finite old price %s accepted", value)
			}
			_, err = productFromBSON(bson.M{"price": int64(1), "accessories": bson.A{bson.M{"price": n}}})
			if err == nil {
				t.Fatalf("product with non-finite accessory price %s accepted", value)
			}
		})
	}
}

func TestVariantDerivedColorOmitsUnknownHex(t *testing.T) {
	product, err := productFromBSON(bson.M{
		"price": int64(1), "variants": bson.A{bson.M{"sku": "S-1", "color": "Đen", "size": "M"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(product.Colors)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `[{"name":"Đen"}]` {
		t.Fatalf("unknown swatch must omit hex: %s", data)
	}
	p2, err := productFromBSON(bson.M{
		"price": int64(1), "colors": bson.A{bson.M{"name": "Đen", "hex": "#000000"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	data, err = json.Marshal(p2.Colors)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "#000000") {
		t.Fatalf("stored swatch was dropped: %s", data)
	}
}

func TestCatalogTimeoutHonorsParentDeadline(t *testing.T) {
	parent, cancelParent := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancelParent()
	child, cancel := catalogContext(parent)
	defer cancel()
	parentDeadline, _ := parent.Deadline()
	childDeadline, _ := child.Deadline()
	if childDeadline.After(parentDeadline) {
		t.Fatalf("deadline extended: %s > %s", childDeadline, parentDeadline)
	}
	other, cancelOther := catalogContext(context.Background())
	defer cancelOther()
	if deadline, ok := other.Deadline(); !ok || time.Until(deadline) > catalogQueryTimeout {
		t.Fatal("unbounded MongoDB context")
	}
}

func TestCatalogAggregationCountsVisiblePageInOneQuery(t *testing.T) {
	pipeline := productPagePipeline(model.Filter{Page: 2, PageSize: 2, Sort: "price_asc"}, []string{"670000000000000000000001"})
	data, err := bson.MarshalExtJSON(bson.M{"pipeline": pipeline}, false, false)
	if err != nil {
		t.Fatal(err)
	}
	serialized := string(data)
	for _, stage := range []string{`"$match"`, `"$sort"`, `"$lookup"`, `"$graphLookup"`, `"$facet"`, `"$count"`, `"$skip"`} {
		if !strings.Contains(serialized, stage) {
			t.Fatalf("missing %s in %s", stage, serialized)
		}
	}
	facet := pipeline[len(pipeline)-1].(bson.M)["$facet"].(bson.M)
	dataStages := facet["data"].(bson.A)
	if dataStages[0].(bson.M)["$skip"] != int64(2) {
		t.Fatalf("page offset %#v", dataStages)
	}
	detail, err := bson.MarshalExtJSON(bson.M{"pipeline": productDetailPipeline("abc")}, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(detail), `"$graphLookup"`) {
		t.Fatal("detail does not verify category")
	}
}
