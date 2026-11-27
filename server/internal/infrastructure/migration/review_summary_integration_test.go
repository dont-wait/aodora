//go:build integration

package migration

import (
	"context"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func reviewDoc(productID bson.ObjectID, rating int32) bson.M {
	now := time.Now().UTC()
	return bson.M{"_id": bson.NewObjectID(), "product_id": productID, "customer_id": bson.NewObjectID(), "rating": rating, "images": bson.A{}, "created_at": now, "updated_at": now}
}

func readProductSummary(t *testing.T, db *mongo.Database, id bson.ObjectID) (float64, int64) {
	t.Helper()
	var row struct {
		Rating      float64 `bson:"rating"`
		ReviewCount int64   `bson:"review_count"`
	}
	if err := db.Collection("products").FindOne(context.Background(), bson.M{"_id": id}).Decode(&row); err != nil {
		t.Fatal(err)
	}
	return row.Rating, row.ReviewCount
}

func TestReviewSummaryRecomputeMigration(t *testing.T) {
	db, uri, files := checkoutTestDatabase(t)
	prior, all, _ := migrationFilesFor(t, files, "review_summary_recompute")
	if err := Run(uri, prior, "up"); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	reviewed, empty := bson.NewObjectID(), bson.NewObjectID()
	for _, id := range []bson.ObjectID{reviewed, empty} {
		// Stale, hand-entered aggregates that do not match the reviews.
		product := bson.M{
			"_id": id, "name": "Áo review " + id.Hex(), "slug": "review-" + id.Hex(), "category_id": bson.NewObjectID(),
			"price": int64(500000), "stock": int32(1), "is_active": true, "rating": 4.9, "review_count": int32(128),
			"created_at": time.Now(), "updated_at": time.Now(),
		}
		if _, err := db.Collection("products").InsertOne(ctx, product); err != nil {
			t.Fatal(err)
		}
	}
	for _, rating := range []int32{4, 5} {
		if _, err := db.Collection("reviews").InsertOne(ctx, reviewDoc(reviewed, rating)); err != nil {
			t.Fatal(err)
		}
	}
	reviewsBefore, err := db.Collection("reviews").CountDocuments(ctx, bson.M{})
	if err != nil {
		t.Fatal(err)
	}

	if err := Run(uri, all, "up"); err != nil {
		t.Fatal(err)
	}
	if rating, count := readProductSummary(t, db, reviewed); rating != 4.5 || count != 2 {
		t.Fatalf("reviewed product summary = %v/%d, want 4.5/2", rating, count)
	}
	if rating, count := readProductSummary(t, db, empty); rating != 0 || count != 0 {
		t.Fatalf("empty product summary = %v/%d, want 0/0", rating, count)
	}
	if n, _ := db.Collection("reviews").CountDocuments(ctx, bson.M{}); n != reviewsBefore {
		t.Fatalf("migration changed reviews: %d -> %d", reviewsBefore, n)
	}

	// Reviews stay account-only (Q-03): a review without customer_id is rejected.
	guest := reviewDoc(reviewed, 5)
	delete(guest, "customer_id")
	guest["order_id"] = bson.NewObjectID()
	if _, err := db.Collection("reviews").InsertOne(ctx, guest); !isValidationError(err) {
		t.Fatalf("review without customer_id accepted: %v", err)
	}

	if err := Run(uri, all, "down"); err != nil {
		t.Fatal(err)
	}
	if err := Run(uri, all, "up"); err != nil {
		t.Fatalf("reapply after rollback: %v", err)
	}
	if rating, count := readProductSummary(t, db, reviewed); rating != 4.5 || count != 2 {
		t.Fatalf("summary after reapply = %v/%d", rating, count)
	}
}
