//go:build integration_standalone

package mongo

import (
	"context"
	"errors"
	"net/url"
	"os"
	"testing"
	"time"

	apporder "github.com/your-org/aodora/server/internal/application/order"
	domainorder "github.com/your-org/aodora/server/internal/domain/order"
	"github.com/your-org/aodora/server/internal/infrastructure/migration"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func standaloneTestDatabase(t *testing.T) (*mongo.Client, *mongo.Database, string) {
	t.Helper()
	raw := os.Getenv("MONGO_STANDALONE_TEST_URI")
	if raw == "" {
		t.Fatal("MONGO_STANDALONE_TEST_URI must point to standalone mongo; integration tests must not silently skip")
	}
	parsedURI, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse MONGO_STANDALONE_TEST_URI: %v", err)
	}

	name := "aodora_standalone_test_" + bson.NewObjectID().Hex()
	parsedURI.Path = "/" + name
	targetURI := parsedURI.String()

	client, err := mongo.Connect(options.Client().ApplyURI(targetURI))
	if err != nil {
		t.Fatalf("connect to standalone mongo: %v", err)
	}

	db := client.Database(name)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = db.Drop(ctx)
		_ = client.Disconnect(ctx)
	})

	migrationsPath := os.Getenv("MONGO_MIGRATIONS_PATH")
	if migrationsPath == "" {
		migrationsPath = "../../../migrations"
	}
	if err := migration.Run(targetURI, os.DirFS(migrationsPath), "up"); err != nil {
		t.Fatalf("run migrations on standalone test db: %v", err)
	}

	return client, db, targetURI
}

func TestStandaloneTransactionUnavailable(t *testing.T) {
	client, db, _ := standaloneTestDatabase(t)
	ctx := context.Background()

	// 1. Seed active category and active product
	catID := bson.NewObjectID()
	now := time.Now().UTC().Truncate(time.Millisecond)
	_, err := db.Collection("categories").InsertOne(ctx, bson.M{
		"_id":        catID,
		"name":       "Áo Thun Standalone",
		"slug":       "ao-thun-standalone",
		"is_active":  true,
		"created_at": now,
		"updated_at": now,
	})
	if err != nil {
		t.Fatalf("seed category: %v", err)
	}

	prodID := bson.NewObjectID()
	initialStock := int64(10)
	_, err = db.Collection("products").InsertOne(ctx, bson.M{
		"_id":         prodID,
		"name":        "Áo Basic Standalone",
		"slug":        "ao-basic-standalone",
		"category_id": catID,
		"price":       int64(150000),
		"stock":       initialStock,
		"variants": bson.A{
			bson.M{"sku": "SA-M", "size": "M", "color": "Đen", "stock": initialStock},
		},
		"is_active":  true,
		"created_at": now,
		"updated_at": now,
	})
	if err != nil {
		t.Fatalf("seed product: %v", err)
	}

	// Clean any demo seed orders so orders count starts at 0
	if _, err := db.Collection("orders").DeleteMany(ctx, bson.M{}); err != nil {
		t.Fatalf("clear orders: %v", err)
	}

	signer, err := domainorder.NewQuoteSigner([]byte("01234567890123456789012345678901"))
	if err != nil {
		t.Fatalf("new quote signer: %v", err)
	}
	shippingFee, err := domainorder.ParseMoney("17000")
	if err != nil {
		t.Fatalf("parse shipping fee: %v", err)
	}

	orderRepo := NewOrderRepository(client, db)
	svc := apporder.NewService(
		orderRepo,
		orderRepo,
		signer,
		15*time.Minute,
		&shippingFee,
		nil,
		nil,
	)

	checkoutInput := domainorder.CheckoutInput{
		PaymentMethod: "cod",
		ShippingAddress: domainorder.ShippingAddressInput{
			Recipient: "Nguyen Van Test",
			Phone:     "0901234567",
			Line1:     "123 Duong 1",
			City:      "TP Ho Chi Minh",
		},
		Items: []domainorder.OrderItemInput{
			{ProductID: prodID.Hex(), SKU: "SA-M", Quantity: 2},
		},
	}

	quote, err := svc.Quote(ctx, checkoutInput)
	if err != nil {
		t.Fatalf("svc.Quote expected success on standalone, got: %v", err)
	}
	checkoutInput.QuoteToken = quote.QuoteToken

	principal := apporder.Principal{
		CustomerID: bson.NewObjectID().Hex(),
	}
	reqID := "a1b2c3d4-e5f6-4a1b-8c2d-3e4f5a6b7c8d"

	order, created, err := svc.Create(ctx, principal, reqID, checkoutInput)
	if err == nil {
		t.Fatalf("expected ErrOrderUnavailable on standalone mongo, got created order: %+v", order)
	}
	if !errors.Is(err, apporder.ErrOrderUnavailable) {
		t.Fatalf("expected ErrOrderUnavailable, got: %v", err)
	}
	if created {
		t.Fatalf("created should be false on failure")
	}

	// 2. Assert orders count is 0
	count, err := db.Collection("orders").CountDocuments(ctx, bson.M{})
	if err != nil {
		t.Fatalf("count orders: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 orders in database, got: %d", count)
	}

	// 3. Assert product stock remains intact
	var prodDoc bson.M
	if err := db.Collection("products").FindOne(ctx, bson.M{"_id": prodID}).Decode(&prodDoc); err != nil {
		t.Fatalf("find product: %v", err)
	}
	if asInt64(prodDoc["stock"]) != initialStock {
		t.Fatalf("expected stock %d, got: %d", initialStock, asInt64(prodDoc["stock"]))
	}
}
