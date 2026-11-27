//go:build integration

package migration

import (
	"context"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func reviewDoc(productID bson.ObjectID, rating int32) bson.M {
	now := time.Now().UTC()
	return bson.M{"_id": bson.NewObjectID(), "product_id": productID, "rating": rating, "images": bson.A{}, "created_at": now, "updated_at": now}
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

func TestReviewGuestAccessMigration(t *testing.T) {
	db, uri, files := checkoutTestDatabase(t)
	prior, all, _ := migrationFilesFor(t, files, "review_guest_access")
	if err := Run(uri, prior, "up"); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	reviewed, empty := bson.NewObjectID(), bson.NewObjectID()
	for _, id := range []bson.ObjectID{reviewed, empty} {
		product := bson.M{
			"_id": id, "name": "Áo review " + id.Hex(), "slug": "review-" + id.Hex(), "category_id": bson.NewObjectID(),
			"price": int64(500000), "stock": int32(1), "is_active": true, "rating": 4.9, "review_count": int32(128),
			"created_at": time.Now(), "updated_at": time.Now(),
		}
		if _, err := db.Collection("products").InsertOne(ctx, product); err != nil {
			t.Fatal(err)
		}
	}
	customerA, customerB := reviewDoc(reviewed, 4), reviewDoc(reviewed, 5)
	customerA["customer_id"], customerB["customer_id"] = bson.NewObjectID(), bson.NewObjectID()
	for _, doc := range []bson.M{customerA, customerB} {
		if _, err := db.Collection("reviews").InsertOne(ctx, doc); err != nil {
			t.Fatal(err)
		}
	}
	guest := reviewDoc(reviewed, 3)
	guest["order_id"] = bson.NewObjectID()
	if _, err := db.Collection("reviews").InsertOne(ctx, guest); !isValidationError(err) {
		t.Fatalf("old schema accepted a review without customer_id: %v", err)
	}

	if err := Run(uri, all, "up"); err != nil {
		t.Fatal(err)
	}
	// Stored aggregates are recomputed from reviews; no reviews means 0/0.
	if rating, count := readProductSummary(t, db, reviewed); rating != 4.5 || count != 2 {
		t.Fatalf("reviewed product summary = %v/%d, want 4.5/2", rating, count)
	}
	if rating, count := readProductSummary(t, db, empty); rating != 0 || count != 0 {
		t.Fatalf("empty product summary = %v/%d, want 0/0", rating, count)
	}

	if _, err := db.Collection("reviews").InsertOne(ctx, guest); err != nil {
		t.Fatalf("guest review rejected: %v", err)
	}
	// A second guest on a different order may review the same product:
	// missing customer_id must not collide in the account unique index.
	otherGuest := reviewDoc(reviewed, 5)
	otherGuest["order_id"] = bson.NewObjectID()
	if _, err := db.Collection("reviews").InsertOne(ctx, otherGuest); err != nil {
		t.Fatalf("second guest review rejected: %v", err)
	}
	duplicate := reviewDoc(reviewed, 1)
	duplicate["order_id"] = guest["order_id"]
	if _, err := db.Collection("reviews").InsertOne(ctx, duplicate); !mongo.IsDuplicateKeyError(err) {
		t.Fatalf("same order/product created a second review: %v", err)
	}
	duplicate = reviewDoc(reviewed, 1)
	duplicate["customer_id"] = customerA["customer_id"]
	if _, err := db.Collection("reviews").InsertOne(ctx, duplicate); !mongo.IsDuplicateKeyError(err) {
		t.Fatalf("same customer/product created a second review: %v", err)
	}
	if _, err := db.Collection("reviews").InsertOne(ctx, reviewDoc(reviewed, 5)); !isValidationError(err) {
		t.Fatalf("review without customer_id and order_id accepted: %v", err)
	}
	invalid := reviewDoc(reviewed, 6)
	invalid["order_id"] = bson.NewObjectID()
	if _, err := db.Collection("reviews").InsertOne(ctx, invalid); !isValidationError(err) {
		t.Fatalf("rating 6 accepted: %v", err)
	}

	// Rollback is refused while guest reviews exist, and changes nothing.
	if err := Run(uri, all, "down"); err == nil || !strings.Contains(err.Error(), "cannot roll back") {
		t.Fatalf("down with guest reviews: %v", err)
	}
	if _, err := db.Collection("reviews").DeleteMany(ctx, bson.M{"customer_id": bson.M{"$exists": false}}); err != nil {
		t.Fatal(err)
	}
	if err := Run(uri, all, "down"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Collection("reviews").InsertOne(ctx, guest); !isValidationError(err) {
		t.Fatalf("down did not restore the account-only validator: %v", err)
	}
	if err := Run(uri, all, "up"); err != nil {
		t.Fatalf("reapply after rollback: %v", err)
	}
}
