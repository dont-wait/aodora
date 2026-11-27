//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	auth "github.com/your-org/aodora/server/internal/application/auth"
	catalogapp "github.com/your-org/aodora/server/internal/application/catalog"
	apporder "github.com/your-org/aodora/server/internal/application/order"
	reviewapp "github.com/your-org/aodora/server/internal/application/review"
	mongorepo "github.com/your-org/aodora/server/internal/infrastructure/mongo"
	"github.com/your-org/aodora/server/internal/logger"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// Fixture order inserted directly: the review flow only depends on the stored
// owner, status and items, which DW-02/DW-03 already cover end to end.
func seedReviewOrder(t *testing.T, db *mongo.Database, customerID *bson.ObjectID, guestHash, status, recipient string, products ...bson.ObjectID) bson.ObjectID {
	t.Helper()
	id := bson.NewObjectID()
	items := bson.A{}
	for _, product := range products {
		items = append(items, bson.M{"product_id": product, "product_name": "Áo review", "sku": "REV-M", "quantity": int32(1), "unit_price": int64(500000), "fulfillment_mode": "ready"})
	}
	now := time.Now().UTC()
	order := bson.M{
		"_id": id, "order_number": "AD-REV-" + id.Hex()[16:], "items": items,
		"subtotal": int64(500000), "shipping_fee": int64(0), "total": int64(500000),
		"status": status, "payment_method": "cash_at_counter", "payment_status": "paid",
		"shipping_address": bson.M{"recipient": recipient, "phone": "0900000009"},
		"fulfillment_mode": "ready", "created_at": now, "updated_at": now,
	}
	scope := ""
	if customerID != nil {
		order["customer_id"] = *customerID
		scope = "customer:" + customerID.Hex()
	} else {
		order["guest_access_hash"] = guestHash
		scope = "guest:" + guestHash
	}
	order["checkout"] = bson.M{"scope": scope, "request_id": fmt.Sprintf("00000000-0000-4000-8000-%s", id.Hex()[12:]), "request_hash": strings.Repeat("c", 64)}
	if _, err := db.Collection("orders").InsertOne(context.Background(), order); err != nil {
		t.Fatalf("seed order: %v", err)
	}
	return id
}

func TestReviewHTTPAcceptanceIntegration(t *testing.T) {
	client, db := httpIntegrationTestDatabase(t)
	ctx := context.Background()
	for _, col := range []string{"orders", "products", "categories", "customers", "sessions", "reviews"} {
		if _, err := db.Collection(col).DeleteMany(ctx, bson.M{}); err != nil {
			t.Fatal(err)
		}
	}
	category := seedHTTPCategory(t, db, "Áo review", "review-http")
	product := seedHTTPProduct(t, db, category, "Áo review HTTP", "REV-M", 500000, 20)
	// seedHTTPProduct derives slugs from the ObjectID timestamp; keep them distinct.
	if _, err := db.Collection("products").UpdateOne(ctx, bson.M{"_id": product}, bson.M{"$set": bson.M{"slug": "ao-review-" + product.Hex()}}); err != nil {
		t.Fatal(err)
	}
	other := seedHTTPProduct(t, db, category, "Áo khác", "REV-O", 500000, 20)
	var productSlug string
	{
		var row struct {
			Slug string `bson:"slug"`
		}
		if err := db.Collection("products").FindOne(ctx, bson.M{"_id": product}).Decode(&row); err != nil {
			t.Fatal(err)
		}
		productSlug = row.Slug
	}

	orderRepo := mongorepo.NewOrderRepository(client, db)
	orders := apporder.NewService(orderRepo, orderRepo, nil, 15*time.Minute, nil, nil, nil)
	authService := auth.NewService(mongorepo.NewAuthRepository(db), time.Hour)
	guard := NewAuthGuard(authService, AuthCookieConfig{Name: "session", TTL: time.Hour, AllowedOrigin: "https://shop.example"})
	catalog := catalogapp.NewService(mongorepo.NewCatalogRepository(db))
	reviews := reviewapp.NewService(mongorepo.NewReviewRepository(db), orders)
	log := *logger.NewDiscard()
	router := NewRouter(nil, log, WithAuth(guard), WithCatalog(catalog, log), WithReviews(reviews, guard, catalog, orders, log),
		WithAdminCatalog(nil, guard.RequireAdmin, log, "https://shop.example"), WithAdminReviews(reviews, guard, log, "https://shop.example"))

	call := func(method, path, body string, cookie *http.Cookie, headers map[string]string, status int) *httptest.ResponseRecorder {
		t.Helper()
		w := adminOrderRequest(router, method, path, body, cookie, headers)
		if w.Code != status {
			t.Fatalf("%s %s: status %d want %d body=%s", method, path, w.Code, status, w.Body.String())
		}
		return w
	}
	customerIDs := map[string]bson.ObjectID{}
	for _, email := range []string{"review-a@example.com", "review-b@example.com", "review-admin@example.com"} {
		call("POST", "/api/v1/auth/register", fmt.Sprintf(`{"full_name":"Khách %s","email":%q,"password":"password-123456"}`, email[7:8], email), nil, nil, 201)
		var row struct {
			ID bson.ObjectID `bson:"_id"`
		}
		if err := db.Collection("customers").FindOne(ctx, bson.M{"email": email}).Decode(&row); err != nil {
			t.Fatal(err)
		}
		customerIDs[email] = row.ID
	}
	// TEST SEED ONLY: no endpoint accepts a client-supplied admin role.
	if _, err := db.Collection("customers").UpdateOne(ctx, bson.M{"email": "review-admin@example.com"}, bson.M{"$set": bson.M{"role": "admin"}}); err != nil {
		t.Fatal(err)
	}
	login := func(email string) *http.Cookie {
		t.Helper()
		w := call("POST", "/api/v1/auth/login", fmt.Sprintf(`{"email":%q,"password":"password-123456"}`, email), nil, nil, 200)
		return w.Result().Cookies()[0]
	}
	cookieA, cookieB, adminCookie := login("review-a@example.com"), login("review-b@example.com"), login("review-admin@example.com")
	idA, idB := customerIDs["review-a@example.com"], customerIDs["review-b@example.com"]

	deliveredA := seedReviewOrder(t, db, &idA, "", "delivered", "", product)
	deliveredA2 := seedReviewOrder(t, db, &idA, "", "delivered", "", product)
	pendingA := seedReviewOrder(t, db, &idA, "", "pending", "", product)
	shippingA := seedReviewOrder(t, db, &idA, "", "shipping", "", product)
	cancelledA := seedReviewOrder(t, db, &idA, "", "cancelled", "", product)
	otherProductA := seedReviewOrder(t, db, &idA, "", "delivered", "", other)
	deliveredB := seedReviewOrder(t, db, &idB, "", "delivered", "", product)
	guestToken, guestHashHex := generateIntegrationGuestToken()
	guestOrder := seedReviewOrder(t, db, nil, guestHashHex, "delivered", "Trần Bình", product)

	put := func(cookie *http.Cookie, order bson.ObjectID, rating int, status int) *httptest.ResponseRecorder {
		t.Helper()
		body := fmt.Sprintf(`{"order_id":%q,"rating":%d,"content":"Áo đẹp","images":[]}`, order.Hex(), rating)
		return call("PUT", "/api/v1/me/reviews/"+product.Hex(), body, cookie, nil, status)
	}
	summary := func() (float64, int64) {
		t.Helper()
		var row struct {
			Rating      float64 `bson:"rating"`
			ReviewCount int64   `bson:"review_count"`
		}
		if err := db.Collection("products").FindOne(ctx, bson.M{"_id": product}).Decode(&row); err != nil {
			t.Fatal(err)
		}
		return row.Rating, row.ReviewCount
	}
	assertSummary := func(wantRating float64, wantCount int64) {
		t.Helper()
		if rating, count := summary(); rating != wantRating || count != wantCount {
			t.Fatalf("products summary = %v/%d, want %v/%d", rating, count, wantRating, wantCount)
		}
		w := call("GET", "/api/v1/products/"+productSlug+"/reviews?page=1&page_size=50", "", nil, nil, 200)
		var page struct {
			Data struct {
				Items   []map[string]any `json:"items"`
				Total   int64            `json:"total"`
				Average float64          `json:"average"`
			} `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &page); err != nil {
			t.Fatal(err)
		}
		// Stored aggregate must match the public list.
		if page.Data.Total != wantCount || int64(len(page.Data.Items)) != wantCount || page.Data.Average != wantRating {
			t.Fatalf("public list = %d items, total %d, average %v; want %d/%v", len(page.Data.Items), page.Data.Total, page.Data.Average, wantCount, wantRating)
		}
		body := w.Body.String()
		for _, secret := range []string{"customer_id", "order_id", "email", "phone", "0900000009", idA.Hex(), idB.Hex(), guestOrder.Hex(), deliveredA.Hex(), guestToken, guestHashHex} {
			if strings.Contains(body, secret) {
				t.Fatalf("public reviews leaked %q: %s", secret, body)
			}
		}
		detail := call("GET", "/api/v1/products/"+productSlug, "", nil, nil, 200)
		var product struct {
			Data struct {
				Reviews struct {
					Count   int64   `json:"count"`
					Average float64 `json:"average"`
				} `json:"reviews"`
			} `json:"data"`
		}
		if err := json.Unmarshal(detail.Body.Bytes(), &product); err != nil {
			t.Fatal(err)
		}
		if product.Data.Reviews.Count != wantCount || product.Data.Reviews.Average != wantRating {
			t.Fatalf("product detail summary = %+v, want %d/%v", product.Data.Reviews, wantCount, wantRating)
		}
	}
	assertSummary(0, 0)

	t.Run("orders that are not delivered, cancelled or lack the product are refused", func(t *testing.T) {
		for _, order := range []bson.ObjectID{pendingA, shippingA, cancelledA, otherProductA, bson.NewObjectID()} {
			put(cookieA, order, 5, http.StatusForbidden)
		}
		call("PUT", "/api/v1/me/reviews/"+product.Hex(), fmt.Sprintf(`{"order_id":%q,"rating":5,"verified_purchase":true,"customer_id":%q}`, pendingA.Hex(), idA.Hex()), cookieA, nil, http.StatusBadRequest)
		call("PUT", "/api/v1/me/reviews/"+product.Hex(), `{"order_id":"x","rating":5}`, nil, nil, http.StatusUnauthorized)
		assertSummary(0, 0)
	})

	t.Run("rating outside 1-5 is rejected", func(t *testing.T) {
		put(cookieA, deliveredA, 0, http.StatusBadRequest)
		put(cookieA, deliveredA, 6, http.StatusBadRequest)
		assertSummary(0, 0)
	})

	t.Run("4 and 5 aggregate to 4.5 with count 2; resubmitting does not duplicate", func(t *testing.T) {
		put(cookieA, deliveredA, 4, http.StatusOK)
		put(cookieA, deliveredA2, 4, http.StatusOK)
		put(cookieB, deliveredB, 5, http.StatusOK)
		if n, _ := db.Collection("reviews").CountDocuments(ctx, bson.M{"customer_id": idA}); n != 1 {
			t.Fatalf("customer A has %d reviews, want 1", n)
		}
		assertSummary(4.5, 2)
		put(cookieA, deliveredA, 5, http.StatusOK)
		assertSummary(5, 2)
		put(cookieA, deliveredA, 4, http.StatusOK)
		assertSummary(4.5, 2)
	})

	t.Run("customer B cannot review or edit as customer A", func(t *testing.T) {
		put(cookieB, deliveredA, 1, http.StatusForbidden)
		var row struct {
			Rating int64 `bson:"rating"`
		}
		if err := db.Collection("reviews").FindOne(ctx, bson.M{"customer_id": idA}).Decode(&row); err != nil || row.Rating != 4 {
			t.Fatalf("customer A review changed: %+v %v", row, err)
		}
		w := call("GET", "/api/v1/me/reviews?product_id="+product.Hex(), "", cookieB, nil, 200)
		if !strings.Contains(w.Body.String(), deliveredB.Hex()) || strings.Contains(w.Body.String(), deliveredA.Hex()) {
			t.Fatalf("customer B read the wrong review: %s", w.Body.String())
		}
		w = call("GET", "/api/v1/me/reviews/eligible-orders?product_id="+product.Hex(), "", cookieA, nil, 200)
		body := w.Body.String()
		for _, id := range []bson.ObjectID{deliveredA, deliveredA2} {
			if !strings.Contains(body, id.Hex()) {
				t.Fatalf("eligible orders missing %s: %s", id.Hex(), body)
			}
		}
		for _, id := range []bson.ObjectID{pendingA, shippingA, cancelledA, otherProductA, deliveredB} {
			if strings.Contains(body, id.Hex()) {
				t.Fatalf("eligible orders included %s: %s", id.Hex(), body)
			}
		}
		assertSummary(4.5, 2)
	})

	t.Run("concurrent submissions keep one review and an exact aggregate", func(t *testing.T) {
		var wg sync.WaitGroup
		codes := make(chan int, 24)
		for i := 0; i < 8; i++ {
			wg.Add(3)
			go func() { // first submissions racing on the unique index
				defer wg.Done()
				body := fmt.Sprintf(`{"order_id":%q,"rating":5}`, otherProductA.Hex())
				codes <- adminOrderRequest(router, "PUT", "/api/v1/me/reviews/"+other.Hex(), body, cookieA, nil).Code
			}()
			go func(rating int) { // two writers racing on one product aggregate
				defer wg.Done()
				body := fmt.Sprintf(`{"order_id":%q,"rating":%d}`, deliveredA.Hex(), rating)
				codes <- adminOrderRequest(router, "PUT", "/api/v1/me/reviews/"+product.Hex(), body, cookieA, nil).Code
			}(1 + i%5)
			go func(rating int) {
				defer wg.Done()
				body := fmt.Sprintf(`{"order_id":%q,"rating":%d}`, deliveredB.Hex(), rating)
				codes <- adminOrderRequest(router, "PUT", "/api/v1/me/reviews/"+product.Hex(), body, cookieB, nil).Code
			}(5 - i%5)
		}
		wg.Wait()
		close(codes)
		for code := range codes {
			if code != http.StatusOK {
				t.Fatalf("concurrent submission status %d", code)
			}
		}
		if n, _ := db.Collection("reviews").CountDocuments(ctx, bson.M{"product_id": other}); n != 1 {
			t.Fatalf("concurrent first submissions created %d reviews", n)
		}
		var stored []struct {
			Rating int64 `bson:"rating"`
		}
		cur, err := db.Collection("reviews").Find(ctx, bson.M{"product_id": product})
		if err != nil || cur.All(ctx, &stored) != nil || len(stored) != 2 {
			t.Fatalf("reviews after race: %v %v", stored, err)
		}
		want := math.Round(float64(stored[0].Rating+stored[1].Rating)/2*100) / 100
		assertSummary(want, 2)
		// Restore the fixture state expected by the next subtests.
		put(cookieA, deliveredA, 4, http.StatusOK)
		put(cookieB, deliveredB, 5, http.StatusOK)
		if _, err := db.Collection("reviews").DeleteMany(ctx, bson.M{"product_id": other}); err != nil {
			t.Fatal(err)
		}
		assertSummary(4.5, 2)
	})

	t.Run("review status reports slug and reviewed flag per product", func(t *testing.T) {
		path := fmt.Sprintf("/api/v1/me/reviews/status?product_id=%s&product_id=%s&product_id=%s&product_id=bad", product.Hex(), other.Hex(), product.Hex())
		call("GET", path, "", nil, nil, http.StatusUnauthorized)
		call("GET", "/api/v1/me/reviews/status?x=1", "", cookieA, nil, http.StatusBadRequest)
		// Hide the other product: it must come back without a slug.
		if _, err := db.Collection("products").UpdateOne(ctx, bson.M{"_id": other}, bson.M{"$set": bson.M{"is_active": false}}); err != nil {
			t.Fatal(err)
		}
		defer func() {
			if _, err := db.Collection("products").UpdateOne(ctx, bson.M{"_id": other}, bson.M{"$set": bson.M{"is_active": true}}); err != nil {
				t.Fatal(err)
			}
		}()
		w := call("GET", path, "", cookieA, nil, http.StatusOK)
		var got struct {
			Data []reviewapp.ProductReviewStatus `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		want := []reviewapp.ProductReviewStatus{
			{ProductID: product.Hex(), Slug: productSlug, Reviewed: true},
			{ProductID: other.Hex(), Slug: "", Reviewed: false},
			{ProductID: "bad"},
		}
		if fmt.Sprint(got.Data) != fmt.Sprint(want) {
			t.Fatalf("statuses = %+v, want %+v", got.Data, want)
		}
		// Customer B has not reviewed "other"; reviewed flags are per session.
		admin := call("GET", fmt.Sprintf("/api/v1/me/reviews/status?product_id=%s", product.Hex()), "", adminCookie, nil, http.StatusOK)
		if !strings.Contains(admin.Body.String(), `"reviewed":false`) {
			t.Fatalf("admin (no review) status = %s", admin.Body.String())
		}
	})

	t.Run("guests without an account can read but not write reviews", func(t *testing.T) {
		guestHeader := map[string]string{"X-Order-Access-Token": guestToken}
		body := fmt.Sprintf(`{"order_id":%q,"rating":5}`, guestOrder.Hex())
		// Even with the delivered order's access token, writing needs a session.
		call("PUT", "/api/v1/me/reviews/"+product.Hex(), body, nil, guestHeader, http.StatusUnauthorized)
		call("GET", "/api/v1/me/reviews/eligible-orders?product_id="+product.Hex(), "", nil, guestHeader, http.StatusUnauthorized)
		call("PUT", "/api/v1/guest-orders/"+guestOrder.Hex()+"/reviews/"+product.Hex(), `{"rating":5}`, nil, guestHeader, http.StatusNotFound)
		// A signed-in account cannot claim a guest order either.
		call("PUT", "/api/v1/me/reviews/"+product.Hex(), body, cookieA, nil, http.StatusForbidden)
		if n, _ := db.Collection("reviews").CountDocuments(ctx, bson.M{"order_id": guestOrder}); n != 0 {
			t.Fatalf("guest order has %d reviews", n)
		}
		// assertSummary reads the public list and product detail without a session.
		assertSummary(4.5, 2)
	})

	t.Run("admin lists and deletes reviews; aggregate follows the public list", func(t *testing.T) {
		call("GET", "/api/v1/admin/reviews", "", cookieA, nil, http.StatusForbidden)
		call("GET", "/api/v1/admin/reviews", "", nil, nil, http.StatusForbidden)
		call("GET", "/api/v1/admin/reviews?rating=9", "", adminCookie, nil, http.StatusBadRequest)
		w := call("GET", "/api/v1/admin/reviews?page_size=50", "", adminCookie, nil, http.StatusOK)
		var page struct {
			Data  []reviewapp.AdminReview `json:"data"`
			Total int64                   `json:"total"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &page); err != nil {
			t.Fatal(err)
		}
		if page.Total != 2 || len(page.Data) != 2 {
			t.Fatalf("admin list = %+v", page)
		}
		for _, secret := range []string{"0900000009", "guest_access_hash", guestHashHex, "email", "customer_id"} {
			if strings.Contains(w.Body.String(), secret) {
				t.Fatalf("admin list leaked %q", secret)
			}
		}
		for _, item := range page.Data {
			if item.ProductName != "Áo review HTTP" || item.OrderNumber == "" {
				t.Fatalf("admin item missing context: %+v", item)
			}
		}
		w = call("GET", "/api/v1/admin/reviews?rating=5", "", adminCookie, nil, http.StatusOK)
		if err := json.Unmarshal(w.Body.Bytes(), &page); err != nil || page.Total != 1 || page.Data[0].Rating != 5 {
			t.Fatalf("rating filter = %+v (%v)", page, err)
		}
		target := page.Data[0].ID

		call("DELETE", "/api/v1/admin/reviews/"+target, "", cookieA, nil, http.StatusForbidden)
		call("DELETE", "/api/v1/admin/reviews/"+target, "", adminCookie, nil, http.StatusOK)
		assertSummary(4, 1)
		call("DELETE", "/api/v1/admin/reviews/"+target, "", adminCookie, nil, http.StatusNotFound)
		call("DELETE", "/api/v1/admin/reviews/not-an-id", "", adminCookie, nil, http.StatusNotFound)

		// Deleting every review returns the aggregate to 0/0.
		w = call("GET", "/api/v1/admin/reviews", "", adminCookie, nil, http.StatusOK)
		if err := json.Unmarshal(w.Body.Bytes(), &page); err != nil {
			t.Fatal(err)
		}
		for _, item := range page.Data {
			call("DELETE", "/api/v1/admin/reviews/"+item.ID, "", adminCookie, nil, http.StatusOK)
		}
		assertSummary(0, 0)
	})
}
