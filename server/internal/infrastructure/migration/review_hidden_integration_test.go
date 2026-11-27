//go:build integration

package migration

import (
	"context"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestReviewHiddenMigration(t *testing.T) {
	db, uri, files := checkoutTestDatabase(t)
	_, all, _ := migrationFilesFor(t, files, "review_hidden")
	if err := Run(uri, all, "up"); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	hidden := reviewDoc(bson.NewObjectID(), 5)
	hidden["hidden"] = true
	if _, err := db.Collection("reviews").InsertOne(ctx, hidden); err != nil {
		t.Fatalf("hidden review rejected: %v", err)
	}
	bad := reviewDoc(bson.NewObjectID(), 4)
	bad["hidden"] = "yes"
	if _, err := db.Collection("reviews").InsertOne(ctx, bad); !isValidationError(err) {
		t.Fatalf("non-boolean hidden accepted: %v", err)
	}
	// The previous validator does not forbid the field, so rollback succeeds
	// with hidden reviews present and reapplying works.
	if err := Run(uri, all, "down"); err != nil {
		t.Fatal(err)
	}
	if err := Run(uri, all, "up"); err != nil {
		t.Fatal(err)
	}
}
