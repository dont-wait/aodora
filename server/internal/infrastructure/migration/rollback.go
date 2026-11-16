package migration

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// The caller holds the migration lock and reads from primary. Application
// writers must still be stopped: an advisory migration lock does not fence them.
func checkDownCompatibility(ctx context.Context, db *mongo.Database, data []byte) error {
	var commands []bson.M
	if err := bson.UnmarshalExtJSON(data, true, &commands); err != nil {
		return fmt.Errorf("decode rollback commands: %w", err)
	}
	for _, command := range commands {
		collection, ok := command["collMod"].(string)
		if !ok || command["validator"] == nil || command["validationAction"] == "warn" {
			continue
		}
		count, err := db.Collection(collection).CountDocuments(ctx, bson.M{"$nor": bson.A{command["validator"]}}, options.Count().SetLimit(1))
		if err != nil {
			return fmt.Errorf("check rollback compatibility for %s: %w", collection, err)
		}
		if count != 0 {
			return fmt.Errorf("cannot roll back: collection %s contains documents incompatible with the previous validator; migrate those documents explicitly before retrying (no data or schema changed)", collection)
		}
	}
	return nil
}
