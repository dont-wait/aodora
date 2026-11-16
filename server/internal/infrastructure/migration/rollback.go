package migration

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/source"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// collMod does not validate documents already stored in a collection. Before
// a rollback installs a stricter validator, reject incompatible data rather
// than leaving historical documents impossible to update. Run migrations with
// application writes stopped: this preflight and the DDL are not one transaction.
func checkDownCompatibility(databaseURL string, migrations source.Driver, runner *migrate.Migrate) error {
	version, dirty, err := runner.Version()
	if errors.Is(err, migrate.ErrNilVersion) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read migration version: %w", err)
	}
	if dirty {
		return fmt.Errorf("cannot roll back dirty migration %d", version)
	}
	file, _, err := migrations.ReadDown(version)
	if err != nil {
		return fmt.Errorf("read down migration %d: %w", version, err)
	}
	defer func() { _ = file.Close() }()
	data, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("read rollback commands: %w", err)
	}
	var commands []bson.M
	if err := bson.UnmarshalExtJSON(data, true, &commands); err != nil {
		return fmt.Errorf("decode rollback commands: %w", err)
	}
	var checks []bson.M
	for _, command := range commands {
		if _, ok := command["collMod"].(string); ok && command["validator"] != nil && command["validationAction"] != "warn" {
			checks = append(checks, command)
		}
	}
	if len(checks) == 0 {
		return nil
	}
	uri, err := url.Parse(databaseURL)
	if err != nil {
		return fmt.Errorf("parse rollback database URL: %w", err)
	}
	query := uri.Query()
	for key := range query {
		if strings.HasPrefix(key, "x-") {
			query.Del(key) // Options understood only by the migrate driver.
		}
	}
	uri.RawQuery = query.Encode()
	client, err := mongo.Connect(options.Client().ApplyURI(uri.String()))
	if err != nil {
		return fmt.Errorf("connect rollback preflight: %w", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = client.Disconnect(ctx)
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db := client.Database(strings.TrimPrefix(uri.Path, "/"))
	for _, command := range checks {
		collection := command["collMod"].(string)
		count, err := db.Collection(collection).CountDocuments(ctx,
			bson.M{"$nor": bson.A{command["validator"]}}, options.Count().SetLimit(1))
		if err != nil {
			return fmt.Errorf("check rollback compatibility for %s: %w", collection, err)
		}
		if count != 0 {
			return fmt.Errorf("cannot roll back migration %d: collection %s contains documents incompatible with the previous validator; migrate those documents explicitly before retrying (no data or schema changed)", version, collection)
		}
	}
	return nil
}
