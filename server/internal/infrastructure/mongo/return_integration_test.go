//go:build integration

package mongo

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	apporder "github.com/your-org/aodora/server/internal/application/order"
	returnsapp "github.com/your-org/aodora/server/internal/application/returns"
	domainorder "github.com/your-org/aodora/server/internal/domain/order"
	dr "github.com/your-org/aodora/server/internal/domain/returns"
)

func TestReturnRequestLifecycle(t *testing.T) {
	client, db, _ := orderTestDatabase(t)
	ctx := context.Background()
	repo := NewOrderRepository(client, db)
	catID := seedCategory(t, db, "Áo dài", "ao-dai", nil, true)
	prodID := seedProduct(t, db, catID, "Áo dài lụa", "ao-dai-lua", int64(1200000), 5, bson.A{
		bson.M{"sku": "AD-M", "size": "M", "color": "Đỏ", "stock": int64(5)},
	}, true)
	custID := bson.NewObjectID().Hex()
	svc := newTestService(t, client, db, repo, "", nil, nil)
	input := domainorder.CheckoutInput{PaymentMethod: "cash_at_counter", ShippingAddress: domainorder.ShippingAddressInput{Recipient: "Trần Thị Mai", Phone: "0912345678"},
		Items: []domainorder.OrderItemInput{{ProductID: prodID.Hex(), SKU: "AD-M", Quantity: 1}}}
	quote, err := svc.Quote(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	input.QuoteToken = quote.QuoteToken
	order, _, err := svc.Create(ctx, apporder.Principal{CustomerID: custID}, "a1b2c3d4-e5f6-4a1b-8c2d-3e4f5a6b7c8d", input)
	if err != nil {
		t.Fatal(err)
	}
	admin := apporder.NewAdminService(repo)
	if _, err := admin.SetStatus(ctx, order.ID, "confirmed"); err != nil {
		t.Fatal(err)
	}
	if _, err := admin.SetStatus(ctx, order.ID, domainorder.StatusDelivered); !errors.Is(err, apporder.ErrStateConflict) {
		t.Fatalf("the shop must not confirm receipt for the buyer: %v", err)
	}
	if _, err := svc.ConfirmReceived(ctx, apporder.Viewer{CustomerID: bson.NewObjectID().Hex()}, order.ID); !errors.Is(err, apporder.ErrNotFound) {
		t.Fatalf("another customer confirmed: %v", err)
	}
	// The buyer confirms receipt at the counter: delivered and paid in one step.
	delivered, err := svc.ConfirmReceived(ctx, apporder.Viewer{CustomerID: custID}, order.ID)
	if err != nil {
		t.Fatal(err)
	}
	if delivered.Status != domainorder.StatusDelivered || delivered.PaymentStatus != domainorder.PaymentPaid || delivered.PaidAt == nil ||
		delivered.DeliveredAt == nil || time.Since(*delivered.DeliveredAt) > time.Minute {
		t.Fatalf("receipt not recorded: %+v", delivered)
	}
	if again, err := svc.ConfirmReceived(ctx, apporder.Viewer{CustomerID: custID}, order.ID); err != nil || !again.DeliveredAt.Equal(*delivered.DeliveredAt) {
		t.Fatalf("second confirm must be a no-op: %v", err)
	}

	prefix := "http://cdn/uploads/returns/"
	returns := returnsapp.NewService(NewReturnRepository(repo), repo, func(id string) string { return prefix + id + "/" })
	in := returnsapp.Input{OrderID: order.ID, Items: []returnsapp.ItemInput{{SKU: "AD-M", Quantity: 1}}, Reason: dr.ReasonDefect, Note: "Đường may bị bung",
		Evidence: []returnsapp.EvidenceInput{{URL: prefix + custID + "/2026/11/a.jpg"}, {URL: prefix + custID + "/2026/11/b.mp4"}}}
	req, err := returns.Create(ctx, custID, in)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	// The unique partial index blocks a second open request even if the service check is raced.
	store := NewReturnRepository(repo)
	dup := req
	dup.ID = ""
	if _, err := store.Create(ctx, dup); !errors.Is(err, returnsapp.ErrConflict) {
		t.Fatalf("duplicate open request: %v", err)
	}
	for _, step := range []returnsapp.TransitionInput{{To: dr.StatusApproved}, {To: dr.StatusCollecting, Note: "Shipper lấy hàng chiều nay"}, {To: dr.StatusCollected, Restock: true}, {To: dr.StatusRefunded}} {
		if req2, err := returns.AdminTransition(ctx, req.ID, step); err != nil {
			t.Fatalf("%s: %v", step.To, err)
		} else {
			req = *req2
		}
	}
	if len(req.History) != 5 || !req.Restocked {
		t.Fatalf("%+v", req)
	}
	var product bson.M
	if err := db.Collection("products").FindOne(ctx, bson.M{"_id": prodID}).Decode(&product); err != nil {
		t.Fatal(err)
	}
	if v := asMap(asSlice(product["variants"])[0]); asInt64(v["stock"]) != 5 {
		t.Fatalf("variant stock after restock = %v", v["stock"])
	}
	final, _ := repo.FindByID(ctx, order.ID)
	if final.PaymentStatus != domainorder.PaymentRefunded || final.RefundedAt == nil {
		t.Fatalf("order not refunded: %s", final.PaymentStatus)
	}
	// Closed requests leave the open index; with a refund the order cannot be returned again.
	e, err := returns.Eligibility(ctx, custID, order.ID)
	if err != nil || e.Eligible || len(e.Requests) != 1 {
		t.Fatalf("eligibility %+v %v", e, err)
	}
	// The validator rejects a request without evidence.
	bad := bson.M{"order_id": prodID, "order_number": "x", "customer_id": prodID, "items": bson.A{bson.M{"product_id": prodID, "sku": "x", "quantity": 1, "unit_price": 1}},
		"reason": "defect", "evidence": bson.A{}, "status": "requested", "refund_amount": 1, "history": bson.A{}, "delivered_at": time.Now(), "created_at": time.Now(), "updated_at": time.Now()}
	if _, err := db.Collection("return_requests").InsertOne(ctx, bad); err == nil {
		t.Fatal("validator accepted a request without evidence")
	}
}
