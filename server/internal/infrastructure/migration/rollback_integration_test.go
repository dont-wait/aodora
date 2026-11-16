//go:build integration

package migration

import (
	"context"
	"strings"
	"testing"
	"testing/fstest"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Different versions, collection and field names prove that preflight follows
// the down file for the current database version, rather than order-specific rules.
func TestRollbackDiscoversValidatorForCurrentVersion(t *testing.T) {
	db, uri, _ := checkoutTestDatabase(t)
	files := fstest.MapFS{
		"000041_initial.up.json": {Data: []byte(`[
			{"create":"rollback_fixture","validator":{"$jsonSchema":{"required":["label"]}}}
		]`)},
		"000041_initial.down.json": {Data: []byte(`[{"drop":"rollback_fixture"}]`)},
		"000078_relax_label.up.json": {Data: []byte(`[
			{"collMod":"rollback_fixture","validator":{"$jsonSchema":{"bsonType":"object"}},"validationAction":"error"}
		]`)},
		"000078_relax_label.down.json": {Data: []byte(`[
			{"collMod":"rollback_fixture","validator":{"$jsonSchema":{"required":["label"]}},"validationAction":"error"}
		]`)},
	}
	if err := Run(uri, files, "up"); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	result, err := db.Collection("rollback_fixture").InsertOne(ctx, bson.M{"other": "preserve this document"})
	if err != nil {
		t.Fatal(err)
	}
	if err := Run(uri, files, "down"); err == nil || !strings.Contains(err.Error(), "collection rollback_fixture contains documents incompatible") {
		t.Fatalf("expected dynamically discovered validation failure: %v", err)
	}
	// Repair the fixture explicitly; the migration must not rewrite it for us.
	if _, err := db.Collection("rollback_fixture").UpdateOne(ctx, bson.M{"_id": result.InsertedID}, bson.M{"$set": bson.M{"label": "valid again"}}); err != nil {
		t.Fatal(err)
	}
	if err := Run(uri, files, "down"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Collection("rollback_fixture").InsertOne(ctx, bson.M{"other": "missing label"}); !isValidationError(err) {
		t.Fatalf("previous validator was not restored: %v", err)
	}
	// A down migration without collMod must still work through the same runner.
	if err := Run(uri, files, "down"); err != nil {
		t.Fatal(err)
	}
}
