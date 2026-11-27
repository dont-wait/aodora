//go:build integration

package migration

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func runMigrationFile(t *testing.T, ctx context.Context, dbRun func(context.Context, any) error, path string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var wrapper struct {
		Commands []bson.D `bson:"c"`
	}
	if err := bson.UnmarshalExtJSON([]byte(`{"c":`+string(raw)+`}`), false, &wrapper); err != nil {
		t.Fatal(err)
	}
	for _, command := range wrapper.Commands {
		if err := dbRun(ctx, command); err != nil {
			t.Fatal(err)
		}
	}
}

func asDoc(v any) bson.M {
	switch d := v.(type) {
	case bson.M:
		return d
	case bson.D:
		out := bson.M{}
		for _, e := range d {
			out[e.Key] = e.Value
		}
		return out
	}
	panic("not a document")
}

func TestSizeStandardMigrationUpAndDown(t *testing.T) {
	db, _ := uuidTestDatabase(t)
	ctx := context.Background()
	products := db.Collection("products")
	_, err := products.InsertMany(ctx, []any{
		bson.M{"_id": "old", "sku": "AD1", "sizes": bson.A{"AA", "A", "B", "C", "D"}, "variants": bson.A{
			bson.M{"sku": "AD1AA", "size": "AA", "color": "Đỏ", "stock": 3},
			bson.M{"sku": "AD1D", "size": "D", "color": "Đỏ", "stock": 0},
			bson.M{"sku": "AD1B", "size": "B", "color": "Xanh", "stock": 1},
		}},
		bson.M{"_id": "unordered", "sizes": bson.A{"C", "A"}, "variants": bson.A{bson.M{"sku": "U-C", "size": "C"}, bson.M{"sku": "U-A", "size": "A"}}},
		bson.M{"_id": "nosize", "sizes": bson.A{}, "variants": bson.A{bson.M{"sku": "N1", "size": ""}}},
		bson.M{"_id": "novariants"},
		bson.M{"_id": "already", "sizes": bson.A{"M", "XL", "Freesize"}, "variants": bson.A{bson.M{"sku": "AL-M", "size": "M"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	run := func(ctx context.Context, command any) error { return db.RunCommand(ctx, command).Err() }
	dir, err := filepath.Glob("../../../migrations/*_size_standard")
	if err != nil || len(dir) != 1 {
		t.Fatalf("size_standard migration directory: %v %v", dir, err)
	}
	read := func(id string) bson.M {
		var doc bson.M
		if err := products.FindOne(ctx, bson.M{"_id": id}).Decode(&doc); err != nil {
			t.Fatal(err)
		}
		return doc
	}
	variantSizes := func(doc bson.M) []string {
		out := []string{}
		if list, ok := doc["variants"].(bson.A); ok {
			for _, item := range list {
				out = append(out, asDoc(item)["size"].(string))
			}
		}
		return out
	}
	sizes := func(doc bson.M) []string {
		out := []string{}
		if list, ok := doc["sizes"].(bson.A); ok {
			for _, item := range list {
				out = append(out, item.(string))
			}
		}
		return out
	}

	runMigrationFile(t, ctx, run, filepath.Join(dir[0], "up.json"))
	old := read("old")
	if got, want := sizes(old), []string{"XS", "S", "M", "L", "XL"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("up sizes = %v, want %v", got, want)
	}
	if got, want := variantSizes(old), []string{"XS", "XL", "M"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("up variant sizes = %v, want %v", got, want)
	}
	if v := asDoc(old["variants"].(bson.A)[0]); v["sku"] != "AD1AA" || v["color"] != "Đỏ" || v["stock"] != int32(3) {
		t.Fatalf("variant fields other than size must be untouched: %v", v)
	}
	if got, want := sizes(read("unordered")), []string{"S", "L"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("unordered sizes = %v, want %v", got, want)
	}
	if got := variantSizes(read("nosize")); !reflect.DeepEqual(got, []string{""}) {
		t.Fatalf("empty size changed: %v", got)
	}
	if _, has := read("novariants")["variants"]; has {
		t.Fatal("documents without variants must stay without variants")
	}
	if got, want := sizes(read("already")), []string{"M", "XL", "Freesize"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("standard sizes touched: %v", got)
	}

	runMigrationFile(t, ctx, run, filepath.Join(dir[0], "down.json"))
	if got, want := sizes(read("old")), []string{"AA", "A", "B", "C", "D"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("down sizes = %v, want %v", got, want)
	}
	if got, want := variantSizes(read("old")), []string{"AA", "D", "B"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("down variant sizes = %v, want %v", got, want)
	}
}
