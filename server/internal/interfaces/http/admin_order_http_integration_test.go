//go:build integration

package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	auth "github.com/your-org/aodora/server/internal/application/auth"
	app "github.com/your-org/aodora/server/internal/application/order"
	domain "github.com/your-org/aodora/server/internal/domain/order"
	mongorepo "github.com/your-org/aodora/server/internal/infrastructure/mongo"
	"github.com/your-org/aodora/server/internal/logger"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestAdminOrderHTTPEndToEndIntegration(t *testing.T) {
	client, db := httpIntegrationTestDatabase(t)
	ctx := context.Background()
	for _, col := range []string{"orders", "products", "categories", "customers", "sessions"} {
		if _, err := db.Collection(col).DeleteMany(ctx, bson.M{}); err != nil {
			t.Fatal(err)
		}
	}
	category := seedHTTPCategory(t, db, "Áo admin HTTP", "admin-http")
	product := seedHTTPProduct(t, db, category, "Áo admin HTTP", "HTTP-M", 500000, 20)
	repo := mongorepo.NewOrderRepository(client, db)
	authRepo := mongorepo.NewAuthRepository(db)
	authService := auth.NewService(authRepo, time.Hour)
	guard := NewAuthGuard(authService, AuthCookieConfig{Name: "session", TTL: time.Hour, AllowedOrigin: "https://shop.example"})
	signer, err := domain.NewQuoteSigner(bytes.Repeat([]byte("s"), 32))
	if err != nil {
		t.Fatal(err)
	}
	fee, err := domain.ParseMoney("17000")
	if err != nil {
		t.Fatal(err)
	} // Test fee only, not an operating policy.
	orders := app.NewService(repo, repo, signer, 15*time.Minute, &fee, nil, nil)
	log := *logger.NewDiscard()
	router := NewRouter(nil, log, WithAuth(guard), WithOrders(orders, guard, log), WithAdminCatalog(nil, guard.RequireAdmin, log, "https://shop.example"), WithAdminOrders(app.NewAdminService(repo), guard.RequireAdmin, log, "https://shop.example"))
	request := func(method, path, body string, cookie *http.Cookie, headers map[string]string, status int) *httptest.ResponseRecorder {
		t.Helper()
		w := adminOrderRequest(router, method, path, body, cookie, headers)
		if w.Code != status {
			t.Fatalf("%s %s: status %d want %d response=%s", method, path, w.Code, status, w.Body.String())
		}
		if strings.HasPrefix(path, "/api/v1/admin/orders") {
			if w.Header().Get("Cache-Control") != "no-store" {
				t.Fatal("admin response missing no-store")
			}
			for _, key := range []string{"guest_access_hash", "checkout", "scope", "request_id", "request_hash"} {
				if strings.Contains(w.Body.String(), key) {
					t.Fatalf("admin leaked %s", key)
				}
			}
		}
		return w
	}
	for _, email := range []string{"admin-http@example.com", "customer-http@example.com"} {
		request("POST", "/api/v1/auth/register", fmt.Sprintf(`{"full_name":"HTTP Test","email":%q,"password":"password-123456"}`, email), nil, nil, 201)
	}
	// TEST SEED ONLY: elevate the registered account directly in this isolated DB.
	// No endpoint accepts a client-supplied admin role.
	res, err := db.Collection("customers").UpdateOne(ctx, bson.M{"email": "admin-http@example.com"}, bson.M{"$set": bson.M{"role": "admin"}})
	if err != nil || res.MatchedCount != 1 {
		t.Fatalf("seed admin role: %v", err)
	}
	login := func(email string) *http.Cookie {
		t.Helper()
		w := request("POST", "/api/v1/auth/login", fmt.Sprintf(`{"email":%q,"password":"password-123456"}`, email), nil, nil, 200)
		cookies := w.Result().Cookies()
		if len(cookies) != 1 {
			t.Fatal("missing session cookie")
		}
		return cookies[0]
	}
	adminCookie := login("admin-http@example.com")
	customerCookie := login("customer-http@example.com")
	guestToken, _ := generateIntegrationGuestToken()
	guestHeader := map[string]string{"X-Order-Access-Token": guestToken}
	create := func(method, key string, quantity int64) string {
		t.Helper()
		body := fmt.Sprintf(`{"payment_method":%q,"shipping_address":{"recipient":"HTTP Test","phone":"0912345678","line1":"123 Test","city":"Test"},"items":[{"product_id":%q,"sku":"HTTP-M","quantity":%d}]}`, method, product.Hex(), quantity)
		quote := request("POST", "/api/v1/orders/quote", body, nil, guestHeader, 200)
		var q struct {
			Data struct {
				QuoteToken string `json:"quote_token"`
			} `json:"data"`
		}
		if err := json.Unmarshal(quote.Body.Bytes(), &q); err != nil {
			t.Fatal(err)
		}
		var input map[string]json.RawMessage
		if err := json.Unmarshal([]byte(body), &input); err != nil {
			t.Fatal(err)
		}
		input["quote_token"], err = json.Marshal(q.Data.QuoteToken)
		if err != nil {
			t.Fatal(err)
		}
		payload, err := json.Marshal(input)
		if err != nil {
			t.Fatal(err)
		}
		headers := map[string]string{"X-Order-Access-Token": guestToken, "Idempotency-Key": key}
		w := request("POST", "/api/v1/orders", string(payload), nil, headers, 201)
		var created struct {
			Data orderDTO `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
			t.Fatal(err)
		}
		if created.Data.Status != "pending" || created.Data.PaymentStatus != "unpaid" {
			t.Fatal("initial state")
		}
		return created.Data.ID
	}
	readAdmin := func(w *httptest.ResponseRecorder) adminOrderDTO {
		t.Helper()
		var payload struct {
			Data adminOrderDTO `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
			t.Fatal(err)
		}
		return payload.Data
	}
	mutation := func(id, suffix, body string) adminOrderDTO {
		t.Helper()
		method := "PATCH"
		if suffix == "cancel" || suffix == "restock" {
			method = "POST"
		}
		w := request(method, "/api/v1/admin/orders/"+id+"/"+suffix, body, adminCookie, map[string]string{"Origin": "https://shop.example"}, 200)
		return readAdmin(w)
	}
	stock := func(want int64) {
		t.Helper()
		var p struct {
			Stock    int64 `bson:"stock"`
			Variants []struct {
				Stock int64 `bson:"stock"`
			} `bson:"variants"`
		}
		if err := db.Collection("products").FindOne(ctx, bson.M{"_id": product}).Decode(&p); err != nil {
			t.Fatal(err)
		}
		if p.Stock != want || len(p.Variants) != 1 || p.Variants[0].Stock != want {
			t.Fatalf("stock=%d variant=%v want=%d", p.Stock, p.Variants, want)
		}
	}
	counterID := create("cash_at_counter", "b0000000-0000-4000-8000-000000000001", 1)
	stock(19)
	assertAdminHTTP(t, request("GET", "/api/v1/admin/orders", "", customerCookie, nil, 403), 403, "ADMIN_REQUIRED", "")
	assertAdminHTTP(t, request("GET", "/api/v1/admin/orders", "", nil, nil, 403), 403, "ADMIN_REQUIRED", "")
	list := request("GET", "/api/v1/admin/orders?status=pending&payment_status=unpaid&page=1&page_size=200", "", adminCookie, nil, 200)
	var page struct {
		Data        []adminOrderDTO `json:"data"`
		Total, Page int64
		PageSize    int64 `json:"page_size"`
	}
	if err := json.Unmarshal(list.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if len(page.Data) != 1 || page.Total != 1 || page.Page != 1 || page.PageSize != 100 || page.Data[0].ID != counterID || !page.Data[0].IsGuest {
		t.Fatal("filtered admin list")
	}
	for _, field := range []string{"status", "payment_status"} {
		w := request("GET", "/api/v1/admin/orders?"+field+"=bad", "", adminCookie, nil, 400)
		assertAdminHTTP(t, w, 400, "INVALID_FIELD", field)
	}
	assertAdminHTTP(t, request("GET", "/api/v1/admin/orders/"+bson.NewObjectID().Hex(), "", adminCookie, nil, 404), 404, "NOT_FOUND", "")
	confirmed := mutation(counterID, "status", `{"status":"confirmed"}`)
	if confirmed.PaymentStatus != "unpaid" {
		t.Fatal("confirm changed payment")
	}
	assertAdminHTTP(t, request("PATCH", "/api/v1/admin/orders/"+counterID+"/status", `{"status":"shipping"}`, adminCookie, nil, 409), 409, "ORDER_STATE_CONFLICT", "")
	// The shop cannot confirm receipt; the buyer (here a guest) does, which also records payment.
	assertAdminHTTP(t, request("PATCH", "/api/v1/admin/orders/"+counterID+"/status", `{"status":"delivered"}`, adminCookie, nil, 409), 409, "ORDER_STATE_CONFLICT", "")
	received := request("POST", "/api/v1/orders/"+counterID+"/received", "", nil, guestHeader, 200)
	if !strings.Contains(received.Body.String(), `"status":"delivered"`) || !strings.Contains(received.Body.String(), `"payment_status":"paid"`) {
		t.Fatalf("guest receipt: %s", received.Body.String())
	}
	request("POST", "/api/v1/orders/"+counterID+"/received", "", nil, nil, 404)
	paid := mutation(counterID, "payment", `{"payment_status":"paid"}`)
	if paid.Status != "delivered" || paid.PaidAt == nil {
		t.Fatal("payment changed delivery")
	}
	guestRead := request("GET", "/api/v1/orders/"+counterID, "", nil, guestHeader, 200)
	var guest struct {
		Data orderDTO `json:"data"`
	}
	if err := json.Unmarshal(guestRead.Body.Bytes(), &guest); err != nil {
		t.Fatal(err)
	}
	if guest.Data.Status != "delivered" || guest.Data.PaymentStatus != "paid" {
		t.Fatal("guest did not see admin changes")
	}
	codID := create("cod", "b0000000-0000-4000-8000-000000000002", 2)
	stock(17)
	mutation(codID, "status", `{"status":"confirmed"}`)
	mutation(codID, "status", `{"status":"shipping"}`)
	cancelled := mutation(codID, "cancel", "")
	if cancelled.Status != "cancelled" || cancelled.InventoryRestock != "awaiting_return" || cancelled.PaymentStatus != "unpaid" {
		t.Fatal("shipping cancellation")
	}
	stock(17)
	restocked := mutation(codID, "restock", `{}`)
	if restocked.InventoryRestock != "restocked" || restocked.InventoryRestockedAt == nil {
		t.Fatal("restock marker")
	}
	stock(19)
	mutation(codID, "cancel", `{}`)
	mutation(codID, "restock", "")
	stock(19)
	mutation(codID, "payment", `{"payment_status":"paid"}`)
	refunded := mutation(codID, "payment", `{"payment_status":"refunded"}`)
	if refunded.Status != "cancelled" || refunded.RefundedAt == nil {
		t.Fatal("refund changed order state")
	}
	assertAdminHTTP(t, request("PATCH", "/api/v1/admin/orders/"+codID+"/payment", `{"payment_status":"paid"}`, adminCookie, nil, 409), 409, "PAYMENT_STATE_CONFLICT", "")
	detail := readAdmin(request("GET", "/api/v1/admin/orders/"+codID, "", adminCookie, nil, 200))
	if detail.PaymentStatus != "refunded" || detail.InventoryRestock != "restocked" {
		t.Fatal("persisted admin state")
	}
	stock(19)
}
