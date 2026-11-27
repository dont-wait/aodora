//go:build integration

package main

import (
	"context"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/your-org/aodora/server/internal/infrastructure/migration"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	mongooptions "go.mongodb.org/mongo-driver/v2/mongo/options"
)

// The generated batch must pass the real collection validators (customers
// require password_hash; orders require a customer:/guest: checkout scope and
// a UUID request id), so insert it into a freshly migrated database.
func TestGeneratedBatchPassesSchemaValidators(t *testing.T) {
	raw := os.Getenv("MONGO_TEST_URI")
	if raw == "" {
		t.Fatal("MONGO_TEST_URI is required for integration tests")
	}
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	u.Path = "/aodora_seedorders_test_" + bson.NewObjectID().Hex()
	client, err := mongo.Connect(mongooptions.Client().ApplyURI(u.String()))
	if err != nil {
		t.Fatal(err)
	}
	db := client.Database(strings.TrimPrefix(u.Path, "/"))
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		_ = db.Drop(ctx)
		_ = client.Disconnect(ctx)
	})
	if err := migration.Run(u.String(), os.DirFS("../../migrations"), "up"); err != nil {
		t.Fatal(err)
	}
	for _, cod := range []*int64{nil, ptr(int64(30_000))} {
		_, b := fixture(t, cod)
		ctx := context.Background()
		if _, err := db.Collection("customers").InsertMany(ctx, b.Customers); err != nil {
			t.Fatalf("customers rejected: %v", err)
		}
		if _, err := db.Collection("orders").InsertMany(ctx, b.Orders); err != nil {
			t.Fatalf("orders rejected: %v", err)
		}
		if _, err := db.Collection("orders").DeleteMany(ctx, bson.M{"seed_tag": seedTag}); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Collection("customers").DeleteMany(ctx, bson.M{"seed_tag": seedTag}); err != nil {
			t.Fatal(err)
		}
	}
}

func ptr[T any](v T) *T { return &v }
