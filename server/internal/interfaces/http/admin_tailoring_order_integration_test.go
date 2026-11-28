//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	auth "github.com/your-org/aodora/server/internal/application/auth"
	apporder "github.com/your-org/aodora/server/internal/application/order"
	apptailoring "github.com/your-org/aodora/server/internal/application/tailoring"
	domainorder "github.com/your-org/aodora/server/internal/domain/order"
	mongorepo "github.com/your-org/aodora/server/internal/infrastructure/mongo"
	"github.com/your-org/aodora/server/internal/logger"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestAdminTailoringCustomOrderIntegration(t *testing.T) {
	client, db := httpIntegrationTestDatabase(t)
	ctx := context.Background()
	for _, col := range []string{"tailoring_requests", "orders", "products", "categories", "customers", "sessions"} {
		if _, err := db.Collection(col).DeleteMany(ctx, bson.M{}); err != nil {
			t.Fatal(err)
		}
	}
	category := seedHTTPCategory(t, db, "Áo may đo", "may-do")
	product := seedHTTPProduct(t, db, category, "Áo dài may đo", "MD-M", 900000, 20)
	time.Sleep(1100 * time.Millisecond) // seed helper derives slug from the ObjectID timestamp (1 s resolution)
	hidden := seedHTTPProduct(t, db, category, "Áo ẩn", "MD-H", 900000, 5)
	if _, err := db.Collection("products").UpdateOne(ctx, bson.M{"_id": hidden}, bson.M{"$set": bson.M{"is_active": false}}); err != nil {
		t.Fatal(err)
	}
	stockOf := func() int64 {
		var doc struct {
			Stock    int64 `bson:"stock"`
			Variants []struct {
				Stock int64 `bson:"stock"`
			} `bson:"variants"`
		}
		if err := db.Collection("products").FindOne(ctx, bson.M{"_id": product}).Decode(&doc); err != nil {
			t.Fatal(err)
		}
		return doc.Stock*1000 + doc.Variants[0].Stock
	}
	baseStock := stockOf()

	tailoringRepo := mongorepo.NewTailoringRepository(db)
	orderRepo := mongorepo.NewOrderRepository(client, db)
	authRepo := mongorepo.NewAuthRepository(db)
	guard := NewAuthGuard(auth.NewService(authRepo, time.Hour), AuthCookieConfig{Name: "session", TTL: time.Hour, AllowedOrigin: "https://shop.example"})
	fee := mustHTTPMoney(t, "17000") // test fee only, not an operating policy
	log := *logger.NewDiscard()
	router := NewRouter(nil, log, WithAuth(guard), WithTailoring(apptailoring.NewService(tailoringRepo), guard, log),
		WithOrders(apporder.NewService(orderRepo, orderRepo, mustQuoteSigner(t), 15*time.Minute, &fee, nil, nil), guard, log),
		WithAdminCatalog(nil, guard.RequireAdmin, log, "https://shop.example"),
		WithAdminTailoring(apptailoring.NewAdminService(tailoringRepo), guard.RequireAdmin, log, "https://shop.example"),
		WithAdminTailoringOrders(apptailoring.NewCustomOrderService(tailoringRepo, tailoringRepo, mongorepo.NewTailoringOrderRepository(client, db), &fee, apporder.DefaultOrderNumber), guard.RequireAdmin, log, "https://shop.example"),
		WithAdminOrders(apporder.NewAdminService(orderRepo), guard.RequireAdmin, log, "https://shop.example"))

	do := func(method, path, body string, cookie *http.Cookie, status int) []byte {
		t.Helper()
		w := adminOrderRequest(router, method, path, body, cookie, nil)
		if w.Code != status {
			t.Fatalf("%s %s: status %d want %d response=%s", method, path, w.Code, status, w.Body.String())
		}
		return w.Body.Bytes()
	}
	code := func(raw []byte) string {
		var e struct {
			Error apiError `json:"error"`
		}
		_ = json.Unmarshal(raw, &e)
		return e.Error.Code
	}
	for _, email := range []string{"admin-co@example.com", "cust-co@example.com", "other-co@example.com"} {
		do("POST", "/api/v1/auth/register", fmt.Sprintf(`{"full_name":"CO Test","email":%q,"password":"password-123456"}`, email), nil, 201)
	}
	if res, err := db.Collection("customers").UpdateOne(ctx, bson.M{"email": "admin-co@example.com"}, bson.M{"$set": bson.M{"role": "admin"}}); err != nil || res.MatchedCount != 1 {
		t.Fatal("seed admin")
	}
	login := func(email string) *http.Cookie {
		w := adminOrderRequest(router, "POST", "/api/v1/auth/login", fmt.Sprintf(`{"email":%q,"password":"password-123456"}`, email), nil, nil)
		if w.Code != 200 || len(w.Result().Cookies()) != 1 {
			t.Fatalf("login %s failed", email)
		}
		return w.Result().Cookies()[0]
	}
	admin, customer, other := login("admin-co@example.com"), login("cust-co@example.com"), login("other-co@example.com")
	var customerID string
	{
		var c struct {
			ID bson.ObjectID `bson:"_id"`
		}
		if err := db.Collection("customers").FindOne(ctx, bson.M{"email": "cust-co@example.com"}).Decode(&c); err != nil {
			t.Fatal(err)
		}
		customerID = c.ID.Hex()
	}

	tr := "/api/v1/admin/tailoring-requests/"
	submit := func(body string, cookie *http.Cookie) string {
		var out struct{ Data struct{ ID string } }
		if err := json.Unmarshal(do("POST", "/api/v1/tailoring-requests", body, cookie, 201), &out); err != nil {
			t.Fatal(err)
		}
		return out.Data.ID
	}
	allCM := `{"measurements":{"body":"1m62","neck_cm":34.5,"bust_cm":86,"waist_cm":65.25,"hips_cm":92,"shoulder_cm":37,"sleeve_cm":56,"garment_length_cm":138}}`
	confirm := func(id string) {
		do("PATCH", tr+id, allCM, admin, 200)
		do("PATCH", tr+id+"/status", `{"status":"contacted"}`, admin, 200)
		do("POST", tr+id+"/confirm-measurements", `{"confirmed":true}`, admin, 200)
	}

	custReq := submit(fmt.Sprintf(`{"full_name":"Khách Tài Khoản","phone":"0911111111","product_id":%q}`, product.Hex()), customer)
	guestReq := submit(`{"full_name":"Khách Vãng Lai","phone":"0922222222"}`, nil)

	orderBody := func(extra string) string {
		return fmt.Sprintf(`{"unit_price":1500000.50,"payment_method":"cash_at_counter"%s}`, extra)
	}
	// Authorization.
	do("POST", tr+custReq+"/order", orderBody(""), customer, 403)
	do("POST", tr+custReq+"/order", orderBody(""), nil, 403)

	// Not orderable yet: submitted, then contacted with incomplete measurements.
	if c := code(do("POST", tr+custReq+"/order", orderBody(""), admin, 409)); c != "TAILORING_STATE_CONFLICT" {
		t.Fatalf("submitted: %s", c)
	}
	do("PATCH", tr+custReq+"/status", `{"status":"contacted"}`, admin, 200)
	do("PATCH", tr+custReq, `{"measurements":{"waist_cm":65.25}}`, admin, 200)
	do("POST", tr+custReq+"/order", orderBody(""), admin, 409) // still contacted
	if c := code(do("POST", tr+custReq+"/confirm-measurements", `{"confirmed":true}`, admin, 409)); c != "MEASUREMENT_INCOMPLETE" {
		t.Fatalf("confirm: %s", c)
	}
	do("PATCH", tr+custReq, allCM, admin, 200)
	do("POST", tr+custReq+"/confirm-measurements", `{"confirmed":true}`, admin, 200)

	// Validation: nothing is written.
	for name, body := range map[string]string{
		"price zero":      `{"unit_price":0,"payment_method":"cash_at_counter"}`,
		"price negative":  `{"unit_price":-1,"payment_method":"cash_at_counter"}`,
		"price text":      `{"unit_price":"abc","payment_method":"cash_at_counter"}`,
		"cod no address":  `{"unit_price":1000,"payment_method":"cod"}`,
		"cod no city":     `{"unit_price":1000,"payment_method":"cod","shipping_address":{"line1":"1 A"}}`,
		"bank transfer":   `{"unit_price":1000,"payment_method":"bank_transfer"}`,
		"other product":   fmt.Sprintf(`{"unit_price":1000,"payment_method":"cash_at_counter","product_id":%q}`, bson.NewObjectID().Hex()),
		"qty zero":        `{"unit_price":1000,"payment_method":"cash_at_counter","quantity":0}`,
		"client customer": `{"unit_price":1000,"payment_method":"cash_at_counter","customer_id":"x"}`,
	} {
		w := adminOrderRequest(router, "POST", tr+custReq+"/order", body, admin, nil)
		if w.Code != 400 {
			t.Fatalf("%s: expected 400, got %d %s", name, w.Code, w.Body.String())
		}
	}
	if n, _ := db.Collection("orders").CountDocuments(ctx, bson.M{}); n != 0 {
		t.Fatalf("validation failures wrote %d orders", n)
	}

	// Happy path: account request, cash at counter, quantity 2 (exact decimal math).
	var created struct {
		Data    adminOrderDTO `json:"data"`
		Created bool          `json:"created"`
	}
	if err := json.Unmarshal(do("POST", tr+custReq+"/order", orderBody(`,"quantity":2,"note":"may gấp"`), admin, 201), &created); err != nil {
		t.Fatal(err)
	}
	o := created.Data
	if !created.Created || o.FulfillmentMode != "custom" || o.TailoringRequestID != custReq || o.CustomerID != customerID || o.IsGuest ||
		o.Status != "pending" || o.PaymentStatus != "unpaid" || o.PaymentMethod != "cash_at_counter" ||
		o.Subtotal.String() != "3000001" || o.ShippingFee.String() != "0" || o.Total.String() != "3000001" || o.Note != "may gấp" {
		t.Fatalf("order: %+v", o)
	}
	if len(o.Items) != 1 || o.Items[0].FulfillmentMode != "custom" || o.Items[0].ProductID != product.Hex() || o.Items[0].Quantity != 2 ||
		o.Items[0].UnitPrice.String() != "1500000.5" || o.Items[0].SKU != "CUSTOM" || o.Items[0].Measurements == nil || *o.Items[0].Measurements.WaistCM != 65.25 {
		t.Fatalf("items: %+v", o.Items)
	}
	mp := o.MeasurementProfile
	if mp == nil || *mp.Body != "1m62" || *mp.NeckCM != 34.5 || *mp.BustCM != 86 || *mp.WaistCM != 65.25 || *mp.HipsCM != 92 || *mp.ShoulderCM != 37 || *mp.SleeveCM != 56 || *mp.GarmentLengthCM != 138 {
		t.Fatalf("measurement_profile: %+v", mp)
	}
	if stockOf() != baseStock {
		t.Fatal("custom order changed stock")
	}

	// Replay: same request again returns the same order; different content conflicts.
	var again struct {
		Data    adminOrderDTO `json:"data"`
		Created bool          `json:"created"`
	}
	if err := json.Unmarshal(do("POST", tr+custReq+"/order", orderBody(`,"quantity":2,"note":"khác ghi chú"`), admin, 200), &again); err != nil {
		t.Fatal(err)
	}
	if again.Created || again.Data.ID != o.ID {
		t.Fatalf("replay: %+v", again)
	}
	if c := code(do("POST", tr+custReq+"/order", `{"unit_price":1,"payment_method":"cash_at_counter","quantity":2}`, admin, 409)); c != "CUSTOM_ORDER_EXISTS" {
		t.Fatalf("different price: %s", c)
	}
	if n, _ := db.Collection("orders").CountDocuments(ctx, bson.M{"tailoring_request_id": mustOID(t, custReq)}); n != 1 {
		t.Fatalf("expected 1 order, got %d", n)
	}

	// Editing the request later never changes the order (snapshot is independent).
	do("PATCH", tr+custReq, `{"notes":"sửa sau","assigned_staff":"Lan"}`, admin, 200)
	do("PATCH", tr+custReq+"/status", `{"status":"in_progress"}`, admin, 200)
	if _, err := db.Collection("tailoring_requests").UpdateOne(ctx, bson.M{"_id": mustOID(t, custReq)}, bson.M{"$set": bson.M{"measurements.waist_cm": 99.0}}); err != nil {
		t.Fatal(err)
	}
	var after struct{ Data adminOrderDTO }
	if err := json.Unmarshal(do("GET", "/api/v1/admin/orders/"+o.ID, "", admin, 200), &after); err != nil {
		t.Fatal(err)
	}
	if *after.Data.MeasurementProfile.WaistCM != 65.25 || *after.Data.Items[0].Measurements.WaistCM != 65.25 || after.Data.Note != "may gấp" || !after.Data.UpdatedAt.Equal(o.UpdatedAt) {
		t.Fatalf("order changed after request edit: %+v", after.Data)
	}
	// Replay still works after the request moved to in_progress.
	do("POST", tr+custReq+"/order", orderBody(`,"quantity":2`), admin, 200)

	// Admin order list shows custom; owner sees it, another account does not.
	var list struct {
		Data []adminOrderDTO `json:"data"`
	}
	if err := json.Unmarshal(do("GET", "/api/v1/admin/orders?status=pending", "", admin, 200), &list); err != nil || len(list.Data) != 1 || list.Data[0].FulfillmentMode != "custom" || list.Data[0].ID != o.ID {
		t.Fatalf("admin list: %v %+v", err, list.Data)
	}
	if body := do("GET", "/api/v1/orders", "", customer, 200); !strings.Contains(string(body), o.ID) {
		t.Fatalf("customer list lacks order: %s", body)
	}
	do("GET", "/api/v1/orders/"+o.ID, "", customer, 200)
	do("GET", "/api/v1/orders/"+o.ID, "", other, 404)
	do("GET", "/api/v1/orders/"+o.ID, "", nil, 404)

	// Order business flow reuses #20: counter order confirm -> deliver -> paid.
	do("PATCH", "/api/v1/admin/orders/"+o.ID+"/status", `{"status":"confirmed"}`, admin, 200)
	do("PATCH", "/api/v1/admin/orders/"+o.ID+"/status", `{"status":"delivered"}`, admin, 409)
	do("POST", "/api/v1/orders/"+o.ID+"/received", "", other, 404)
	do("POST", "/api/v1/orders/"+o.ID+"/received", "", customer, 200)
	do("PATCH", "/api/v1/admin/orders/"+o.ID+"/payment", `{"payment_status":"paid"}`, admin, 200)

	// Guest request: COD needs address, adds the configured fee, no customer_id.
	confirm(guestReq)
	do("POST", tr+guestReq+"/order", `{"unit_price":2000000,"payment_method":"cod"}`, admin, 400)
	if c := code(do("POST", tr+guestReq+"/order", `{"unit_price":2000000,"payment_method":"cod","product_id":"`+hidden.Hex()+`","shipping_address":{"line1":"1 A","city":"Huế"}}`, admin, 400)); c != "INVALID_FIELD" {
		t.Fatalf("hidden product: %s", c)
	}
	do("POST", tr+guestReq+"/order", `{"unit_price":2000000,"payment_method":"cod","shipping_address":{"line1":"1 A","city":"Huế"}}`, admin, 400) // no product anywhere

	// Concurrent creates: exactly one order, the others replay it.
	body := fmt.Sprintf(`{"product_id":%q,"unit_price":2000000.25,"payment_method":"cod","shipping_address":{"line1":"1 Lê Lợi","city":"Huế"}}`, product.Hex())
	const workers = 8
	var wg sync.WaitGroup
	codes := make([]int, workers)
	ids := make([]string, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			w := adminOrderRequest(router, "POST", tr+guestReq+"/order", body, admin, nil)
			codes[i] = w.Code
			var r struct{ Data struct{ ID string } }
			_ = json.Unmarshal(w.Body.Bytes(), &r)
			ids[i] = r.Data.ID
		}(i)
	}
	wg.Wait()
	created201 := 0
	for i := range codes {
		if codes[i] == 201 {
			created201++
		} else if codes[i] != 200 {
			t.Fatalf("worker %d: status %d", i, codes[i])
		}
		if ids[i] == "" || ids[i] != ids[0] {
			t.Fatalf("workers returned different orders: %v", ids)
		}
	}
	if created201 != 1 {
		t.Fatalf("expected exactly one 201, got %d (%v)", created201, codes)
	}
	if n, _ := db.Collection("orders").CountDocuments(ctx, bson.M{"tailoring_request_id": mustOID(t, guestReq)}); n != 1 {
		t.Fatalf("expected 1 guest order, got %d", n)
	}
	var guestOrder struct{ Data adminOrderDTO }
	if err := json.Unmarshal(do("GET", "/api/v1/admin/orders/"+ids[0], "", admin, 200), &guestOrder); err != nil {
		t.Fatal(err)
	}
	g := guestOrder.Data
	if !g.IsGuest || g.CustomerID != "" || g.ShippingFee.String() != "17000" || g.Total.String() != "2017000.25" || g.ShippingAddress.Recipient != "Khách Vãng Lai" || g.ShippingAddress.Phone != "0922222222" {
		t.Fatalf("guest order: %+v", g)
	}
	// Guest has no credential: no anonymous read, no list leak.
	do("GET", "/api/v1/orders/"+ids[0], "", nil, 404)
	do("GET", "/api/v1/orders/"+ids[0], "", customer, 404)

	// Cancelling a pending custom order never restocks.
	var cancelled struct{ Data adminOrderDTO }
	if err := json.Unmarshal(do("POST", "/api/v1/admin/orders/"+ids[0]+"/cancel", "", admin, 200), &cancelled); err != nil {
		t.Fatal(err)
	}
	if cancelled.Data.Status != "cancelled" || cancelled.Data.InventoryRestock != "not_applicable" {
		t.Fatalf("cancel: %+v", cancelled.Data)
	}
	if stockOf() != baseStock {
		t.Fatal("cancel changed stock")
	}
	// One order per request, even after cancel: no silent second order.
	do("POST", tr+guestReq+"/order", body, admin, 200)
	if n, _ := db.Collection("orders").CountDocuments(ctx, bson.M{"tailoring_request_id": mustOID(t, guestReq)}); n != 1 {
		t.Fatalf("second order after cancel: %d", n)
	}
	var _ = domainorder.ZeroMoney
}

func mustOID(t *testing.T, hex string) bson.ObjectID {
	t.Helper()
	id, err := bson.ObjectIDFromHex(hex)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func mustHTTPMoney(t *testing.T, s string) domainorder.Money {
	t.Helper()
	m, err := domainorder.ParseMoney(s)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func mustQuoteSigner(t *testing.T) *domainorder.QuoteSigner {
	t.Helper()
	s, err := domainorder.NewQuoteSigner([]byte(strings.Repeat("s", 32)))
	if err != nil {
		t.Fatal(err)
	}
	return s
}
