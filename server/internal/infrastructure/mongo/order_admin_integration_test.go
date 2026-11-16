//go:build integration

package mongo

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	app "github.com/your-org/aodora/server/internal/application/order"
	domain "github.com/your-org/aodora/server/internal/domain/order"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type adminFixture struct {
	db       *mongo.Database
	repo     *OrderRepository
	admin    *app.AdminService
	customer *app.Service
	order    domain.Order
	products []bson.ObjectID
}

func newAdminFixture(t *testing.T, client *mongo.Client, db *mongo.Database, method string) *adminFixture {
	t.Helper()
	ctx := context.Background()
	cat := seedCategory(t, db, "Admin test", "admin-"+bson.NewObjectID().Hex(), nil, true)
	p := seedProduct(t, db, cat, "Variants", "variants-"+bson.NewObjectID().Hex(), int64(100), 30, bson.A{bson.M{"sku": "A", "size": "M", "color": "Black", "stock": int64(10)}, bson.M{"sku": "B", "size": "L", "color": "Black", "stock": int64(20)}}, true)
	p2 := seedProduct(t, db, cat, "Legacy", "legacy-"+bson.NewObjectID().Hex(), int64(200), 15, nil, true)
	if _, err := db.Collection("products").UpdateByID(ctx, p2, bson.M{"$set": bson.M{"sku": "LEGACY"}}); err != nil {
		t.Fatal(err)
	}
	repo := NewOrderRepository(client, db)
	svc := newTestService(t, client, db, repo, "17000", nil, nil)
	input := domain.CheckoutInput{PaymentMethod: method, ShippingAddress: domain.ShippingAddressInput{Recipient: "Admin Test", Phone: "0912345678", Line1: "123 Test", City: "Test City"}, Items: []domain.OrderItemInput{{ProductID: p.Hex(), SKU: "A", Quantity: 2}, {ProductID: p.Hex(), SKU: "B", Quantity: 3}, {ProductID: p2.Hex(), SKU: "LEGACY", Quantity: 4}}}
	q, err := svc.Quote(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	input.QuoteToken = q.QuoteToken
	// Each fixture uses a new account, making the fixed request UUID unique by scope.
	o, created, err := svc.Create(ctx, app.Principal{CustomerID: bson.NewObjectID().Hex()}, "b1b2c3d4-e5f6-4a1b-8c2d-3e4f5a6b7c8d", input)
	if err != nil || !created {
		t.Fatalf("create: %v created=%v", err, created)
	}
	return &adminFixture{db: db, repo: repo, admin: app.NewAdminService(repo), customer: svc, order: o, products: []bson.ObjectID{p, p2}}
}
func (f *adminFixture) stock(t *testing.T, restocked bool) {
	t.Helper()
	want := []int64{25, 11}
	va, vb := int64(8), int64(17)
	if restocked {
		want = []int64{30, 15}
		va, vb = 10, 20
	}
	for i, id := range f.products {
		var doc bson.M
		if err := f.db.Collection("products").FindOne(context.Background(), bson.M{"_id": id}).Decode(&doc); err != nil {
			t.Fatal(err)
		}
		if asInt64(doc["stock"]) != want[i] {
			t.Fatalf("product %d stock=%v want=%d", i, doc["stock"], want[i])
		}
		if i == 0 {
			vs := asSlice(doc["variants"])
			if asInt64(asMap(vs[0])["stock"]) != va || asInt64(asMap(vs[1])["stock"]) != vb {
				t.Fatalf("variant stocks: %v", vs)
			}
		}
		if asInt64(doc["version"]) != 1 {
			t.Fatal("version changed")
		}
		if _, exists := doc["inventory_operations"]; exists {
			t.Fatal("inventory_operations changed")
		}
	}
}
func (f *adminFixture) read(t *testing.T) *domain.Order {
	t.Helper()
	o, err := f.customer.Get(context.Background(), app.Viewer{CustomerID: f.order.CustomerID}, f.order.ID)
	if err != nil {
		t.Fatal(err)
	}
	return o
}
func adminOK(t *testing.T, o *domain.Order, err error) *domain.Order {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
	return o
}
func (f *adminFixture) status(t *testing.T, target string) *domain.Order {
	t.Helper()
	o, e := f.admin.SetStatus(context.Background(), f.order.ID, target)
	return adminOK(t, o, e)
}
func (f *adminFixture) cancel(t *testing.T) *domain.Order {
	t.Helper()
	o, e := f.admin.Cancel(context.Background(), f.order.ID)
	return adminOK(t, o, e)
}
func (f *adminFixture) restock(t *testing.T) *domain.Order {
	t.Helper()
	o, e := f.admin.ConfirmRestock(context.Background(), f.order.ID)
	return adminOK(t, o, e)
}
func parallelAdmin(n int, fn func(int) error) []error {
	start := make(chan struct{})
	errs := make([]error, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) { defer wg.Done(); <-start; errs[i] = fn(i) }(i)
	}
	close(start)
	wg.Wait()
	return errs
}
func assertAdminErrors(t *testing.T, errs []error) {
	t.Helper()
	for _, err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
}
func TestAdminOrderLifecycle(t *testing.T) {
	client, db, _ := orderTestDatabase(t)
	ctx := context.Background()
	for _, method := range []string{"cod", "cash_at_counter"} {
		t.Run(method, func(t *testing.T) {
			f := newAdminFixture(t, client, db, method)
			if _, err := f.admin.ConfirmRestock(ctx, f.order.ID); !errors.Is(err, app.ErrStateConflict) {
				t.Fatal(err)
			}
			f.status(t, "confirmed")
			if method == "cod" {
				f.status(t, "shipping")
			} else {
				if _, err := f.admin.SetStatus(ctx, f.order.ID, "shipping"); !errors.Is(err, app.ErrStateConflict) {
					t.Fatal(err)
				}
			}
			delivered := f.status(t, "delivered")
			repeat := f.status(t, "delivered")
			if !delivered.UpdatedAt.Equal(repeat.UpdatedAt) {
				t.Fatal("retry wrote status")
			}
			if f.read(t).PaymentStatus != "unpaid" {
				t.Fatal("delivery changed payment")
			}
			if _, err := f.admin.Cancel(ctx, f.order.ID); !errors.Is(err, app.ErrStateConflict) {
				t.Fatal(err)
			}
			f.stock(t, false)
		})
	}
	for _, confirmed := range []bool{false, true} {
		t.Run(fmt.Sprintf("early_cancel_%v", confirmed), func(t *testing.T) {
			f := newAdminFixture(t, client, db, "cod")
			if confirmed {
				f.status(t, "confirmed")
			}
			assertAdminErrors(t, parallelAdmin(8, func(int) error { _, err := f.admin.Cancel(ctx, f.order.ID); return err }))
			first := f.read(t)
			for i := 0; i < 3; i++ {
				o := f.cancel(t)
				if !o.UpdatedAt.Equal(first.UpdatedAt) {
					t.Fatal("retry wrote cancellation")
				}
			}
			if first.Status != "cancelled" || first.InventoryRestock != "restocked" || first.PaymentStatus != "unpaid" || first.CancelledAt == nil || first.InventoryRestockedAt == nil {
				t.Fatal("invalid early cancellation")
			}
			f.restock(t)
			f.stock(t, true)
		})
	}
	t.Run("shipping_return", func(t *testing.T) {
		f := newAdminFixture(t, client, db, "cod")
		f.status(t, "confirmed")
		f.status(t, "shipping")
		o := f.cancel(t)
		if o.InventoryRestock != "awaiting_return" || o.InventoryRestockedAt != nil {
			t.Fatal("invalid awaiting return")
		}
		f.stock(t, false)
		assertAdminErrors(t, parallelAdmin(8, func(int) error { _, err := f.admin.ConfirmRestock(ctx, f.order.ID); return err }))
		first := f.read(t)
		for i := 0; i < 3; i++ {
			o = f.restock(t)
			if !o.UpdatedAt.Equal(first.UpdatedAt) {
				t.Fatal("retry wrote restock")
			}
		}
		f.stock(t, true)
	})
	t.Run("payment", func(t *testing.T) {
		f := newAdminFixture(t, client, db, "cod")
		if _, err := f.admin.SetPayment(ctx, f.order.ID, "refunded"); !errors.Is(err, app.ErrPaymentConflict) {
			t.Fatal(err)
		}
		times := make([]time.Time, 8)
		assertAdminErrors(t, parallelAdmin(8, func(i int) error {
			o, err := f.admin.SetPayment(ctx, f.order.ID, "paid")
			if err == nil {
				if o.PaidAt == nil {
					return errors.New("missing paid_at")
				}
				times[i] = *o.PaidAt
			}
			return err
		}))
		for _, ts := range times {
			if !ts.Equal(times[0]) {
				t.Fatal("paid_at overwritten")
			}
		}
		if f.read(t).Status != "pending" {
			t.Fatal("payment changed status")
		}
		f.cancel(t)
		if f.read(t).PaymentStatus != "paid" {
			t.Fatal("cancel refunded automatically")
		}
		o, err := f.admin.SetPayment(ctx, f.order.ID, "refunded")
		adminOK(t, o, err)
		if o.RefundedAt == nil || o.Status != "cancelled" {
			t.Fatal("refund fields")
		}
		repeat, err := f.admin.SetPayment(ctx, f.order.ID, "refunded")
		adminOK(t, repeat, err)
		if !repeat.UpdatedAt.Equal(o.UpdatedAt) {
			t.Fatal("refund retry wrote")
		}
		if _, err := f.admin.SetPayment(ctx, f.order.ID, "paid"); !errors.Is(err, app.ErrPaymentConflict) {
			t.Fatal(err)
		}
	})
	t.Run("payment_and_status", func(t *testing.T) {
		f := newAdminFixture(t, client, db, "cod")
		assertAdminErrors(t, parallelAdmin(2, func(i int) error {
			if i == 0 {
				_, err := f.admin.SetPayment(ctx, f.order.ID, "paid")
				return err
			}
			_, err := f.admin.SetStatus(ctx, f.order.ID, "confirmed")
			return err
		}))
		o := f.read(t)
		if o.Status != "confirmed" || o.PaymentStatus != "paid" || o.PaidAt == nil {
			t.Fatal("lost concurrent update")
		}
	})
}

func TestAdminOrderRaces(t *testing.T) {
	client, db, _ := orderTestDatabase(t)
	ctx := context.Background()
	for _, method := range []string{"cod", "cash_at_counter"} {
		t.Run(method, func(t *testing.T) {
			for i := 0; i < 20; i++ {
				f := newAdminFixture(t, client, db, method)
				f.status(t, "confirmed")
				target := "shipping"
				if method == "cash_at_counter" {
					target = "delivered"
				}
				errs := parallelAdmin(2, func(i int) error {
					if i == 0 {
						_, err := f.admin.SetStatus(ctx, f.order.ID, target)
						return err
					}
					_, err := f.admin.Cancel(ctx, f.order.ID)
					return err
				})
				o := f.read(t)
				switch {
				case o.Status == "cancelled" && o.InventoryRestock == "restocked":
					if !errors.Is(errs[0], app.ErrStateConflict) || errs[1] != nil {
						t.Fatalf("cancel winner: %v", errs)
					}
					f.stock(t, true)
				case method == "cod" && o.Status == "cancelled" && o.InventoryRestock == "awaiting_return":
					assertAdminErrors(t, errs)
					f.stock(t, false)
				case method == "cash_at_counter" && o.Status == "delivered":
					if errs[0] != nil || !errors.Is(errs[1], app.ErrStateConflict) {
						t.Fatalf("delivery winner: %v", errs)
					}
					f.stock(t, false)
				default:
					t.Fatalf("invalid final state: %s %s errors=%v", o.Status, o.InventoryRestock, errs)
				}
			}
		})
	}
}

func TestAdminOrderRestockFailuresAndLegacy(t *testing.T) {
	client, db, _ := orderTestDatabase(t)
	ctx := context.Background()
	for _, mode := range []string{"deleted_product", "deleted_sku", "legacy", "custom", "mixed_custom", "inactive"} {
		t.Run(mode, func(t *testing.T) {
			f := newAdminFixture(t, client, db, "cod")
			id, _ := oid(f.order.ID)
			switch mode {
			case "deleted_product":
				if _, err := db.Collection("products").DeleteOne(ctx, bson.M{"_id": f.products[1]}); err != nil {
					t.Fatal(err)
				}
			case "deleted_sku":
				if _, err := db.Collection("products").UpdateByID(ctx, f.products[0], bson.M{"$pull": bson.M{"variants": bson.M{"sku": "B"}}}); err != nil {
					t.Fatal(err)
				}
			case "legacy":
				if _, err := db.Collection("orders").UpdateByID(ctx, id, bson.M{"$unset": bson.M{"checkout": ""}, "$set": bson.M{"payment_method": "bank_transfer"}}); err != nil {
					t.Fatal(err)
				}
				f.status(t, "confirmed")
				f.status(t, "shipping")
			case "custom":
				if _, err := db.Collection("orders").UpdateByID(ctx, id, bson.M{"$set": bson.M{"items.$[].fulfillment_mode": "custom"}}); err != nil {
					t.Fatal(err)
				}
			case "mixed_custom":
				if _, err := db.Collection("orders").UpdateByID(ctx, id, bson.M{"$set": bson.M{"items.2.fulfillment_mode": "custom"}}); err != nil {
					t.Fatal(err)
				}
			case "inactive":
				if _, err := db.Collection("products").UpdateMany(ctx, bson.M{"_id": bson.M{"$in": f.products}}, bson.M{"$set": bson.M{"is_active": false}}); err != nil {
					t.Fatal(err)
				}
			}
			o, err := f.admin.Cancel(ctx, f.order.ID)
			if mode == "deleted_product" || mode == "deleted_sku" {
				if !errors.Is(err, app.ErrRestockUnavailable) {
					t.Fatal(err)
				}
				if f.read(t).Status != "pending" {
					t.Fatal("failed restock changed order")
				}
				var doc bson.M
				if err := db.Collection("products").FindOne(ctx, bson.M{"_id": f.products[0]}).Decode(&doc); err != nil {
					t.Fatal(err)
				}
				if asInt64(doc["stock"]) != 25 || asInt64(asMap(asSlice(doc["variants"])[0])["stock"]) != 8 {
					t.Fatal("partial restock")
				}
				return
			}
			adminOK(t, o, err)
			if mode == "legacy" || mode == "custom" {
				if o.InventoryRestock != "not_applicable" {
					t.Fatal(o.InventoryRestock)
				}
				f.stock(t, false)
				if _, err := f.admin.ConfirmRestock(ctx, f.order.ID); !errors.Is(err, app.ErrStateConflict) {
					t.Fatal(err)
				}
			} else if mode == "mixed_custom" {
				var variant, legacy bson.M
				if err := db.Collection("products").FindOne(ctx, bson.M{"_id": f.products[0]}).Decode(&variant); err != nil {
					t.Fatal(err)
				}
				if err := db.Collection("products").FindOne(ctx, bson.M{"_id": f.products[1]}).Decode(&legacy); err != nil {
					t.Fatal(err)
				}
				if asInt64(variant["stock"]) != 30 || asInt64(legacy["stock"]) != 11 {
					t.Fatal("mixed custom inventory restored incorrectly")
				}
			} else {
				f.stock(t, true)
			}
		})
	}
}

func TestAdminOrderList(t *testing.T) {
	client, db, _ := orderTestDatabase(t)
	ctx := context.Background()
	f := newAdminFixture(t, client, db, "cod")
	id, _ := oid(f.order.ID)
	var original bson.M
	if err := db.Collection("orders").FindOne(ctx, bson.M{"_id": id}).Decode(&original); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Collection("orders").DeleteMany(ctx, bson.M{}); err != nil {
		t.Fatal(err)
	}
	stamp := time.Now().UTC().Truncate(time.Millisecond)
	var expected []string
	for i := 0; i < 9; i++ {
		doc := bson.M{}
		for k, v := range original {
			doc[k] = v
		}
		newID := bson.NewObjectID()
		doc["_id"] = newID
		doc["order_number"] = fmt.Sprintf("LIST-%d", i)
		doc["created_at"] = stamp
		delete(doc, "checkout")
		if i%2 == 0 {
			delete(doc, "customer_id")
			doc["guest_access_hash"] = fmt.Sprintf("%064x", i+1)
		}
		if i < 6 {
			doc["status"] = "confirmed"
		}
		if i < 3 {
			doc["payment_status"] = "paid"
		}
		if _, err := db.Collection("orders").InsertOne(ctx, doc); err != nil {
			t.Fatal(err)
		}
		expected = append([]string{newID.Hex()}, expected...)
	}
	seen := map[string]bool{}
	for page := int64(1); page <= 3; page++ {
		orders, total, _, _, err := f.admin.List(ctx, app.AdminFilter{Page: page, PageSize: 3})
		if err != nil || total != 9 || len(orders) != 3 {
			t.Fatalf("page %d total=%d count=%d err=%v", page, total, len(orders), err)
		}
		for i, o := range orders {
			if seen[o.ID] || o.ID != expected[int((page-1)*3)+i] {
				t.Fatal("unstable pagination")
			}
			seen[o.ID] = true
		}
	}
	for _, tc := range []struct {
		filter app.AdminFilter
		want   int64
	}{{app.AdminFilter{Status: "confirmed"}, 6}, {app.AdminFilter{PaymentStatus: "paid"}, 3}, {app.AdminFilter{Status: "confirmed", PaymentStatus: "paid"}, 3}, {app.AdminFilter{Status: "pending", PaymentStatus: "paid"}, 0}} {
		orders, total, _, _, err := f.admin.List(ctx, tc.filter)
		if err != nil || total != tc.want || int64(len(orders)) != tc.want {
			t.Fatalf("filter %+v total=%d err=%v", tc.filter, total, err)
		}
	}
	for _, missing := range []string{"invalid", bson.NewObjectID().Hex()} {
		if _, err := f.admin.Get(ctx, missing); !errors.Is(err, app.ErrNotFound) {
			t.Fatal(err)
		}
		if _, err := f.admin.Cancel(ctx, missing); !errors.Is(err, app.ErrNotFound) {
			t.Fatal(err)
		}
	}
	// A directly inserted legacy guest order has no checkout marker and never restores stock.
	legacyID := expected[0]
	o, err := f.admin.Cancel(ctx, legacyID)
	adminOK(t, o, err)
	if o.InventoryRestock != "not_applicable" || o.CustomerID != "" {
		t.Fatal("legacy guest cancellation")
	}
	f.stock(t, false)
}
