//go:build integration

package migration

import (
	"context"
	"errors"
	"io/fs"
	"net/url"
	"os"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/golang-migrate/migrate/v4/source"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func checkoutTestDatabase(t *testing.T) (*mongo.Database, string, fs.FS) {
	t.Helper()
	raw := os.Getenv("MONGO_TEST_URI")
	if raw == "" {
		t.Fatal("MONGO_TEST_URI must point to the isolated replica set; integration tests must not silently skip")
	}
	uri, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	// Never operate on the caller's database. Each test owns a unique database.
	name := "aodora_dw01_test_" + bson.NewObjectID().Hex()
	uri.Path = "/" + name
	client, err := mongo.Connect(options.Client().ApplyURI(uri.String()))
	if err != nil {
		t.Fatal(err)
	}
	db := client.Database(name)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := db.Drop(ctx); err != nil {
			t.Errorf("drop isolated test database: %v", err)
		}
		if err := client.Disconnect(ctx); err != nil {
			t.Errorf("disconnect test database: %v", err)
		}
	})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var hello bson.M
	if err := db.RunCommand(ctx, bson.D{{Key: "hello", Value: 1}}).Decode(&hello); err != nil {
		t.Fatal(err)
	}
	if hello["setName"] == nil || hello["isWritablePrimary"] != true {
		t.Fatalf("tests require a writable replica set: %v", hello)
	}
	path := os.Getenv("MONGO_MIGRATIONS_PATH")
	if path == "" {
		path = "../../../migrations"
	}
	return db, uri.String(), os.DirFS(path)
}

// Discover the migration under test by its identifier, not its numeric slot.
// Later migrations are excluded so rollback always targets checkout itself.
func checkoutMigrationFiles(t *testing.T, all fs.FS) (fs.FS, fs.FS, uint) {
	t.Helper()
	entries, err := fs.ReadDir(all, ".")
	if err != nil {
		t.Fatal(err)
	}
	var target uint
	found := false
	for _, entry := range entries {
		migration, err := source.DefaultParse(entry.Name())
		if err == nil && migration.Identifier == "order_checkout" && migration.Direction == source.Up {
			if found {
				t.Fatal("more than one order_checkout up migration found")
			}
			target, found = migration.Version, true
		}
	}
	if !found {
		t.Fatal("order_checkout up migration not found")
	}
	before, through := fstest.MapFS{}, fstest.MapFS{}
	for _, entry := range entries {
		migration, err := source.DefaultParse(entry.Name())
		if entry.IsDir() || err != nil || migration.Version > target {
			continue
		}
		data, err := fs.ReadFile(all, entry.Name())
		if err != nil {
			t.Fatal(err)
		}
		through[entry.Name()] = &fstest.MapFile{Data: data}
		if migration.Version < target {
			before[entry.Name()] = &fstest.MapFile{Data: data}
		}
	}
	return before, through, target
}

func checkoutOrder() bson.M {
	return bson.M{
		"_id": bson.NewObjectID(), "customer_id": bson.NewObjectID(),
		"items": bson.A{bson.M{
			"product_id": bson.NewObjectID(), "product_name": "Áo dài kiểm thử",
			"sku": "DW01-M", "quantity": int32(1), "unit_price": int64(500000),
			"fulfillment_mode": "ready",
		}},
		"subtotal": int64(500000), "shipping_fee": int64(30000), "total": int64(530000),
		"status": "pending", "payment_method": "cod", "payment_status": "unpaid",
		"shipping_address": bson.M{"recipient": "Nguyễn An", "phone": "0900000001", "line1": "12 Nguyễn Trãi", "city": "TP. Hồ Chí Minh"},
		"fulfillment_mode": "ready", "created_at": time.Now().UTC(), "updated_at": time.Now().UTC(),
	}
}

func counterOrder() bson.M {
	order := checkoutOrder()
	delete(order, "customer_id")
	order["payment_method"] = "cash_at_counter"
	order["shipping_fee"] = int64(0)
	order["total"] = order["subtotal"]
	order["shipping_address"] = bson.M{"recipient": "Nguyễn An", "phone": "0900000001"}
	return order
}

func readOrders(t *testing.T, db *mongo.Database) []bson.M {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cursor, err := db.Collection("orders").Find(ctx, bson.M{}, options.Find().SetSort(bson.D{{Key: "_id", Value: 1}}))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cursor.Close(ctx) }()
	var orders []bson.M
	if err := cursor.All(ctx, &orders); err != nil {
		t.Fatal(err)
	}
	return orders
}

func TestCheckoutMigrationCompatibilityAndRollback(t *testing.T) {
	db, uri, files := checkoutTestDatabase(t)
	prior, all, targetVersion := checkoutMigrationFiles(t, files)
	ctx := context.Background()
	if err := Run(uri, prior, "up"); err != nil {
		t.Fatal(err)
	}
	legacy := checkoutOrder()
	if _, err := db.Collection("orders").InsertOne(ctx, legacy); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Collection("orders").InsertOne(ctx, counterOrder()); !isValidationError(err) {
		t.Fatalf("old schema unexpectedly accepted counter order: %v", err)
	}
	before := readOrders(t, db)
	if err := Run(uri, all, "up"); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, readOrders(t, db)) {
		t.Fatal("up migration changed existing orders")
	}

	valid := []struct {
		name  string
		order bson.M
	}{
		{"legacy account COD", checkoutOrder()},
		{"guest counter missing customer", counterOrder()},
		{"guest COD missing customer", checkoutOrder()},
		{"guest COD null customer", checkoutOrder()},
		{"guest counter null customer", counterOrder()},
		{"legacy bank transfer", checkoutOrder()},
		{"legacy card", checkoutOrder()},
		{"legacy e-wallet", checkoutOrder()},
		{"custom counter", counterOrder()},
	}
	delete(valid[2].order, "customer_id")
	valid[3].order["customer_id"] = nil
	valid[4].order["customer_id"] = nil
	valid[5].order["payment_method"] = "bank_transfer"
	valid[6].order["payment_method"] = "card"
	valid[7].order["payment_method"] = "e_wallet"
	valid[8].order["fulfillment_mode"] = "custom"
	valid[8].order["tailoring_request_id"] = bson.NewObjectID()
	valid[8].order["measurement_profile"] = bson.M{"body": "1m62 / 52kg", "bust_cm": 86.5}
	var inserted []any
	for _, tc := range valid {
		t.Run(tc.name, func(t *testing.T) {
			result, err := db.Collection("orders").InsertOne(ctx, tc.order)
			if err != nil {
				t.Fatal(err)
			}
			inserted = append(inserted, result.InsertedID)
		})
	}
	invalid := []struct {
		name   string
		change func(bson.M)
	}{
		{"COD missing street", func(o bson.M) { delete(o["shipping_address"].(bson.M), "line1") }},
		{"COD missing city", func(o bson.M) { delete(o["shipping_address"].(bson.M), "city") }},
		{"COD null street", func(o bson.M) { o["shipping_address"].(bson.M)["line1"] = nil }},
		{"counter missing phone", func(o bson.M) {
			o["payment_method"] = "cash_at_counter"
			o["shipping_fee"] = 0
			delete(o["shipping_address"].(bson.M), "phone")
		}},
		{"counter missing recipient", func(o bson.M) {
			o["payment_method"] = "cash_at_counter"
			o["shipping_fee"] = 0
			delete(o["shipping_address"].(bson.M), "recipient")
		}},
		{"counter nonzero fee", func(o bson.M) { o["payment_method"] = "cash_at_counter" }},
		{"customer string", func(o bson.M) { o["customer_id"] = "not-an-object-id" }},
		{"unknown payment", func(o bson.M) { o["payment_method"] = "crypto" }},
		{"quantity zero", func(o bson.M) { o["items"].(bson.A)[0].(bson.M)["quantity"] = int32(0) }},
		{"negative measurement", func(o bson.M) { o["measurement_profile"] = bson.M{"bust_cm": -1} }},
	}
	for _, tc := range invalid {
		t.Run(tc.name, func(t *testing.T) {
			order := checkoutOrder()
			tc.change(order)
			if _, err := db.Collection("orders").InsertOne(ctx, order); !isValidationError(err) {
				t.Fatalf("expected schema validation error, got %v", err)
			}
		})
	}
	beforeDown := readOrders(t, db)
	err := Run(uri, all, "down")
	if err == nil || !strings.Contains(err.Error(), "incompatible with the previous validator") {
		t.Fatalf("unsafe rollback was not blocked: %v", err)
	}
	if !reflect.DeepEqual(beforeDown, readOrders(t, db)) {
		t.Fatal("failed rollback changed orders")
	}
	var version struct {
		Version int64 `bson:"version"`
		Dirty   bool  `bson:"dirty"`
	}
	if err := db.Collection("schema_migrations").FindOne(ctx, bson.M{}).Decode(&version); err != nil {
		t.Fatal(err)
	}
	if version.Version != int64(targetVersion) || version.Dirty {
		t.Fatalf("blocked rollback changed migration state: %v", version)
	}
	// Test-owned fixtures only: the runner never deletes/transforms orders.
	if _, err := db.Collection("orders").DeleteMany(ctx, bson.M{"_id": bson.M{"$in": inserted}}); err != nil {
		t.Fatal(err)
	}
	if err := Run(uri, all, "down"); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, readOrders(t, db)) {
		t.Fatal("rollback changed legacy orders")
	}
	if _, err := db.Collection("orders").InsertOne(ctx, counterOrder()); !isValidationError(err) {
		t.Fatalf("rollback did not restore old validator: %v", err)
	}
	if err := Run(uri, all, "up"); err != nil {
		t.Fatalf("reapply after rollback: %v", err)
	}
}

func isValidationError(err error) bool {
	var serverError mongo.ServerError
	return errors.As(err, &serverError) && serverError.HasErrorCode(121)
}

func TestCheckoutReplicaSetCommitsAndAbortsAcrossCollections(t *testing.T) {
	db, uri, files := checkoutTestDatabase(t)
	_, all, _ := checkoutMigrationFiles(t, files)
	if err := Run(uri, all, "up"); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	productID := bson.NewObjectID()
	product := bson.M{
		"_id": productID, "name": "Transaction fixture", "slug": "dw01-transaction-fixture",
		"category_id": bson.NewObjectID(), "price": int64(500000), "stock": int32(1),
		"is_active": true, "created_at": time.Now(), "updated_at": time.Now(),
	}
	if _, err := db.Collection("products").InsertOne(ctx, product); err != nil {
		t.Fatal(err)
	}
	session, err := db.Client().StartSession()
	if err != nil {
		t.Fatal(err)
	}
	defer session.EndSession(ctx)
	order := counterOrder()
	order["items"].(bson.A)[0].(bson.M)["product_id"] = productID
	abort := errors.New("simulated failure after both writes")
	write := func(txCtx context.Context) error {
		result, err := db.Collection("products").UpdateOne(txCtx, bson.M{"_id": productID, "stock": bson.M{"$gte": 1}}, bson.M{"$inc": bson.M{"stock": -1}})
		if err != nil {
			return err
		}
		if result.ModifiedCount != 1 {
			return errors.New("stock was not updated")
		}
		_, err = db.Collection("orders").InsertOne(txCtx, order)
		return err
	}
	_, err = session.WithTransaction(ctx, func(txCtx context.Context) (any, error) {
		if err := write(txCtx); err != nil {
			return nil, err
		}
		return nil, abort
	})
	if !errors.Is(err, abort) {
		t.Fatalf("transaction did not return simulated failure: %v", err)
	}
	assertState := func(wantStock int32, wantOrders int64) {
		t.Helper()
		var got bson.M
		if err := db.Collection("products").FindOne(ctx, bson.M{"_id": productID}).Decode(&got); err != nil {
			t.Fatal(err)
		}
		count, err := db.Collection("orders").CountDocuments(ctx, bson.M{"_id": order["_id"]})
		if err != nil || count != wantOrders || got["stock"] != wantStock {
			t.Fatalf("stock=%v orders=%d err=%v; want stock=%d orders=%d", got["stock"], count, err, wantStock, wantOrders)
		}
	}
	assertState(1, 0)
	_, err = session.WithTransaction(ctx, func(txCtx context.Context) (any, error) { return nil, write(txCtx) })
	if err != nil {
		t.Fatal(err)
	}
	assertState(0, 1)
}
