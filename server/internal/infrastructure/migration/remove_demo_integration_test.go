//go:build integration

package migration

import (
	"context"
	"path/filepath"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// The legacy demo seed (men's shirts, jeans, a midi dress) does not belong in an
// áo dài shop. The UUID migration removes exactly those documents and their
// orders/carts/reviews, leaves everything else alone, and down restores them.
func TestRemoveNonAoDaiDemoUpAndDown(t *testing.T) {
	db, _ := uuidTestDatabase(t)
	ctx := context.Background()
	run := func(ctx context.Context, command any) error { return db.RunCommand(ctx, command).Err() }
	runMigrationFile(t, ctx, run, "../../../migrations/000003_demo_seed.up.json")
	if _, err := db.Collection("products").InsertOne(ctx, bson.M{"_id": "ao-dai", "name": "Áo dài lụa"}); err != nil {
		t.Fatal(err)
	}
	dir, err := filepath.Glob("../../../migrations/*_remove_non_ao_dai_demo")
	if err != nil || len(dir) != 1 {
		t.Fatalf("remove_non_ao_dai_demo migration directory: %v %v", dir, err)
	}
	counts := func() map[string]int64 {
		out := map[string]int64{}
		for _, name := range []string{"categories", "products", "orders", "carts", "reviews", "customers"} {
			n, err := db.Collection(name).CountDocuments(ctx, bson.M{})
			if err != nil {
				t.Fatal(err)
			}
			out[name] = n
		}
		return out
	}
	seeded := counts()

	runMigrationFile(t, ctx, run, filepath.Join(dir[0], "up.json"))
	got := counts()
	want := map[string]int64{"categories": 0, "products": 1, "orders": 0, "carts": 0, "reviews": 0, "customers": seeded["customers"]}
	for name, n := range want {
		if got[name] != n {
			t.Fatalf("after up %s = %d, want %d", name, got[name], n)
		}
	}

	runMigrationFile(t, ctx, run, filepath.Join(dir[0], "down.json"))
	if got := counts(); got["categories"] != seeded["categories"] || got["products"] != seeded["products"] || got["orders"] != seeded["orders"] || got["carts"] != seeded["carts"] || got["reviews"] != seeded["reviews"] {
		t.Fatalf("down did not restore demo seed: %v, want %v", got, seeded)
	}
}
