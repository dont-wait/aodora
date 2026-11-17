//go:build integration_standalone

package mongo

import (
	"context"
	"errors"
	"testing"
	"time"

	app "github.com/your-org/aodora/server/internal/application/order"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestAdminStandaloneAtomicity(t *testing.T) {
	client, db, _ := standaloneTestDatabase(t)
	ctx := context.Background()
	repo := NewOrderRepository(client, db)
	admin := app.NewAdminService(repo)
	now := time.Now().UTC().Truncate(time.Millisecond)
	product := bson.NewObjectID()
	category := bson.NewObjectID()
	if _, err := db.Collection("products").InsertOne(ctx, bson.M{"_id": product, "name": "Standalone", "slug": "admin-standalone", "category_id": category, "price": int64(100), "stock": int64(8), "sku": "SA", "is_active": true, "created_at": now, "updated_at": now}); err != nil {
		t.Fatal(err)
	}
	for _, status := range []string{"pending", "cancelled"} {
		t.Run(status, func(t *testing.T) {
			id := bson.NewObjectID()
			doc := bson.M{"_id": id, "order_number": "SA-" + id.Hex(), "customer_id": id, "status": status, "payment_status": "unpaid", "payment_method": "cod", "subtotal": int64(200), "shipping_fee": int64(0), "total": int64(200), "fulfillment_mode": "ready", "shipping_address": bson.M{"recipient": "Test", "phone": "0912345678", "line1": "123 Test", "city": "Test"}, "items": bson.A{bson.M{"product_id": product, "product_name": "Standalone", "sku": "SA", "quantity": int64(2), "unit_price": int64(100), "fulfillment_mode": "ready"}}, "checkout": bson.M{"scope": "customer:" + id.Hex(), "request_id": "b1b2c3d4-e5f6-4a1b-8c2d-3e4f5a6b7c8d", "request_hash": "0000000000000000000000000000000000000000000000000000000000000000"}, "created_at": now, "updated_at": now}
			if status == "cancelled" {
				doc["inventory_restock"] = "awaiting_return"
			}
			if _, err := db.Collection("orders").InsertOne(ctx, doc); err != nil {
				t.Fatal(err)
			}
			var err error
			if status == "pending" {
				_, err = admin.Cancel(ctx, id.Hex())
			} else {
				_, err = admin.ConfirmRestock(ctx, id.Hex())
			}
			if !errors.Is(err, app.ErrOrderUnavailable) {
				t.Fatalf("expected unavailable: %v", err)
			}
			o, err := admin.Get(ctx, id.Hex())
			if err != nil {
				t.Fatal(err)
			}
			if o.Status != status || !o.UpdatedAt.Equal(now) || o.InventoryRestockedAt != nil {
				t.Fatal("partial order mutation")
			}
			if status == "cancelled" && o.InventoryRestock != "awaiting_return" {
				t.Fatal("restock marker changed")
			}
			var p bson.M
			if err := db.Collection("products").FindOne(ctx, bson.M{"_id": product}).Decode(&p); err != nil {
				t.Fatal(err)
			}
			if asInt64(p["stock"]) != 8 {
				t.Fatal("partial inventory mutation")
			}
			// Single-document state/payment writes still work without transactions.
			if status == "pending" {
				if _, err := admin.SetStatus(ctx, id.Hex(), "confirmed"); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := admin.SetPayment(ctx, id.Hex(), "paid"); err != nil {
				t.Fatal(err)
			}
		})
	}
}
