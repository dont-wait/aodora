//go:build integration

package migration

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

const checkoutAccessIndex = "uq_orders_checkout_scope_request"

func markedOrder(guest bool) bson.M {
	order := checkoutOrder()
	scope := "customer:" + order["customer_id"].(bson.ObjectID).Hex()
	if guest {
		order = counterOrder()
		order["guest_access_hash"] = strings.Repeat("a", 64)
		scope = "guest:" + order["guest_access_hash"].(string)
	}
	requestID, err := newUUID()
	if err != nil {
		panic(err)
	}
	order["checkout"] = bson.M{"scope": scope, "request_id": requestID, "request_hash": strings.Repeat("b", 64)}
	return order
}

func TestCheckoutAccessSchemaAndIdempotencyIndex(t *testing.T) {
	db, uri, files := checkoutTestDatabase(t)
	prior, all, _ := migrationFilesFor(t, files, "order_checkout_access")
	if err := Run(uri, prior, "up"); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	// Existing documents have no checkout marker and keep historical enum values.
	legacy := checkoutOrder()
	legacy["payment_method"] = "bank_transfer"
	if _, err := db.Collection("orders").InsertOne(ctx, legacy); err != nil {
		t.Fatal(err)
	}
	before := readOrders(t, db)
	if err := Run(uri, all, "up"); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, readOrders(t, db)) {
		t.Fatal("access migration rewrote historical orders")
	}
	for _, guest := range []bool{false, true} {
		order := markedOrder(guest)
		if _, err := db.Collection("orders").InsertOne(ctx, order); err != nil {
			t.Fatal(err)
		}
		order["_id"] = bson.NewObjectID()
		if _, err := db.Collection("orders").InsertOne(ctx, order); !mongo.IsDuplicateKeyError(err) {
			t.Fatalf("same scope/key created duplicate order: %v", err)
		}
		if guest {
			order["guest_access_hash"] = strings.Repeat("c", 64)
			order["checkout"].(bson.M)["scope"] = "guest:" + strings.Repeat("c", 64)
		} else {
			order["customer_id"] = bson.NewObjectID()
			order["checkout"].(bson.M)["scope"] = "customer:" + order["customer_id"].(bson.ObjectID).Hex()
		}
		if _, err := db.Collection("orders").InsertOne(ctx, order); err != nil {
			t.Fatalf("same key in different ownership scope rejected: %v", err)
		}
	}
	accountCounter := markedOrder(false)
	accountCounter["payment_method"], accountCounter["shipping_fee"], accountCounter["total"] = "cash_at_counter", int64(0), accountCounter["subtotal"]
	accountCounter["shipping_address"] = bson.M{"recipient": "Khách tại quầy", "phone": "0900000000"}
	guestCOD := markedOrder(true)
	guestCOD["payment_method"], guestCOD["shipping_fee"], guestCOD["total"] = "cod", int64(30000), int64(530000)
	guestCOD["shipping_address"] = bson.M{"recipient": "Khách giao hàng", "phone": "0900000000", "line1": "12 Nguyễn Trãi", "city": "TP. Hồ Chí Minh"}
	for _, order := range []bson.M{accountCounter, guestCOD} {
		if _, err := db.Collection("orders").InsertOne(ctx, order); err != nil {
			t.Fatalf("valid checkout combination rejected: %v", err)
		}
	}
	// Concurrent inserts with one scope/key cannot bypass the unique index.
	template := markedOrder(true)
	encoded, err := bson.Marshal(template)
	if err != nil {
		t.Fatal(err)
	}
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() {
			var order bson.M
			if err := bson.Unmarshal(encoded, &order); err != nil {
				results <- err
				return
			}
			order["_id"] = bson.NewObjectID()
			_, err := db.Collection("orders").InsertOne(ctx, order)
			results <- err
		}()
	}
	success, duplicates := 0, 0
	for i := 0; i < 2; i++ {
		err := <-results
		if err == nil {
			success++
		} else if mongo.IsDuplicateKeyError(err) {
			duplicates++
		} else {
			t.Fatal(err)
		}
	}
	if success != 1 || duplicates != 1 {
		t.Fatalf("concurrent duplicate checkout accepted: successes=%d duplicates=%d", success, duplicates)
	}
	invalid := []struct {
		name   string
		guest  bool
		change func(bson.M)
	}{
		{"new bank transfer", false, func(o bson.M) { o["payment_method"] = "bank_transfer" }},
		{"new card", false, func(o bson.M) { o["payment_method"] = "card" }},
		{"new e-wallet", false, func(o bson.M) { o["payment_method"] = "e_wallet" }},
		{"missing scope", false, func(o bson.M) { delete(o["checkout"].(bson.M), "scope") }},
		{"bad request UUID", false, func(o bson.M) { o["checkout"].(bson.M)["request_id"] = "short" }},
		{"missing request hash", false, func(o bson.M) { delete(o["checkout"].(bson.M), "request_hash") }},
		{"plaintext hash", false, func(o bson.M) { o["checkout"].(bson.M)["request_hash"] = "plaintext" }},
		{"another customer scope", false, func(o bson.M) { o["checkout"].(bson.M)["scope"] = "customer:" + bson.NewObjectID().Hex() }},
		{"customer using guest token", false, func(o bson.M) { o["guest_access_hash"] = strings.Repeat("a", 64) }},
		{"guest missing credential", true, func(o bson.M) { delete(o, "guest_access_hash") }},
		{"guest credential plaintext", true, func(o bson.M) { o["guest_access_hash"] = "plain-token" }},
		{"another guest scope", true, func(o bson.M) { o["checkout"].(bson.M)["scope"] = "guest:" + strings.Repeat("d", 64) }},
	}
	for _, tc := range invalid {
		t.Run(tc.name, func(t *testing.T) {
			o := markedOrder(tc.guest)
			tc.change(o)
			if _, err := db.Collection("orders").InsertOne(ctx, o); !isValidationError(err) {
				t.Fatalf("expected validation rejection, got %v", err)
			}
		})
	}
	// Repeated legacy shapes remain valid; the unique index only covers marked checkout rows.
	for i := 0; i < 2; i++ {
		old := checkoutOrder()
		old["payment_method"] = "card"
		if _, err := db.Collection("orders").InsertOne(ctx, old); err != nil {
			t.Fatal(err)
		}
	}
	withAccess := readOrders(t, db)
	if err := Run(uri, all, "down"); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(withAccess, readOrders(t, db)) {
		t.Fatal("down removed security metadata or changed orders")
	}
	indexes, err := db.Collection("orders").Indexes().List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = indexes.Close(ctx) }()
	var rows []bson.M
	if err := indexes.All(ctx, &rows); err != nil {
		t.Fatal(err)
	}
	for _, index := range rows {
		if index["name"] == checkoutAccessIndex {
			t.Fatal("down retained idempotency index")
		}
	}
	if err := Run(uri, all, "up"); err != nil {
		t.Fatalf("reapply access migration: %v", err)
	}
}
