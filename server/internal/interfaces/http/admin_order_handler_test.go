package httpapi

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
	auth "github.com/your-org/aodora/server/internal/application/auth"
	app "github.com/your-org/aodora/server/internal/application/order"
	domain "github.com/your-org/aodora/server/internal/domain/order"
	"github.com/your-org/aodora/server/internal/logger"
)

type adminOrderStub struct {
	calls              int
	method, id, target string
	filter             app.AdminFilter
	order              domain.Order
	err                error
}

func (s *adminOrderStub) called(method, id, target string) (*domain.Order, error) {
	s.calls++
	s.method = method
	s.id = id
	s.target = target
	return &s.order, s.err
}
func (s *adminOrderStub) Get(_ context.Context, id string) (*domain.Order, error) {
	return s.called("get", id, "")
}
func (s *adminOrderStub) SetStatus(_ context.Context, id, target string) (*domain.Order, error) {
	return s.called("status", id, target)
}
func (s *adminOrderStub) SetPayment(_ context.Context, id, target string) (*domain.Order, error) {
	return s.called("payment", id, target)
}
func (s *adminOrderStub) Cancel(_ context.Context, id string) (*domain.Order, error) {
	return s.called("cancel", id, "")
}
func (s *adminOrderStub) ConfirmRestock(_ context.Context, id string) (*domain.Order, error) {
	return s.called("restock", id, "")
}
func (s *adminOrderStub) List(_ context.Context, f app.AdminFilter) ([]domain.Order, int64, int64, int64, error) {
	s.calls++
	s.method = "list"
	s.filter = f
	return []domain.Order{s.order}, 1, f.Page, f.PageSize, s.err
}

func adminOrderUnitRouter(log zerolog.Logger) (http.Handler, *adminOrderStub, *authHTTPRepository, *http.Cookie) {
	token := strings.Repeat("a", 64)
	hash := sha256.Sum256([]byte(token))
	repo := &authHTTPRepository{user: auth.User{ID: "680000000000000000000001", Role: "admin"}, tokenHash: hex.EncodeToString(hash[:]), expiresAt: time.Now().Add(time.Hour)}
	guard := NewAuthGuard(auth.NewService(repo, time.Hour), AuthCookieConfig{Name: "session"})
	stub := &adminOrderStub{order: domain.Order{ID: "680000000000000000000002", OrderNumber: "ADMIN-TEST", Status: "pending", PaymentMethod: "cod", PaymentStatus: "unpaid"}}
	router := NewRouter(nil, log, WithAdminCatalog(nil, guard.RequireAdmin, log, "https://shop.example"), WithAdminOrders(stub, guard.RequireAdmin, log, "https://shop.example"))
	return router, stub, repo, &http.Cookie{Name: "session", Value: token}
}
func adminOrderRequest(router http.Handler, method, path, body string, cookie *http.Cookie, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if cookie != nil {
		req.AddCookie(cookie)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}
func assertAdminHTTP(t *testing.T, w *httptest.ResponseRecorder, status int, code, field string) {
	t.Helper()
	if w.Code != status {
		t.Fatalf("status=%d want=%d body=%s", w.Code, status, w.Body.String())
	}
	if w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("missing no-store")
	}
	if code != "" {
		var body struct {
			Error apiError `json:"error"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if body.Error.Code != code || body.Error.Field != field {
			t.Fatalf("error=%+v want=%s/%s", body.Error, code, field)
		}
	}
}

func TestAdminOrderAuthorizationAndPreflight(t *testing.T) {
	router, s, repo, cookie := adminOrderUnitRouter(*logger.NewDiscard())
	routes := []struct{ method, path, body string }{{"GET", "/api/v1/admin/orders", ""}, {"GET", "/api/v1/admin/orders/x", ""}, {"PATCH", "/api/v1/admin/orders/x/status", `{"status":"confirmed"}`}, {"PATCH", "/api/v1/admin/orders/x/payment", `{"payment_status":"paid"}`}, {"POST", "/api/v1/admin/orders/x/cancel", ""}, {"POST", "/api/v1/admin/orders/x/restock", ""}}
	for _, route := range routes {
		for _, role := range []string{"guest", "customer"} {
			repo.user.Role = role
			credential := cookie
			if role == "guest" {
				credential = nil
			}
			w := adminOrderRequest(router, route.method, route.path, route.body, credential, nil)
			assertAdminHTTP(t, w, 403, "ADMIN_REQUIRED", "")
		}
	}
	if s.calls != 0 {
		t.Fatal("unauthorized service call")
	}
	repo.user.Role = "admin"
	w := adminOrderRequest(router, "PATCH", "/api/v1/admin/orders/x/status", `{"status":"confirmed"}`, cookie, map[string]string{"Origin": "https://evil.example"})
	assertAdminHTTP(t, w, 403, "ORIGIN_FORBIDDEN", "")
	if s.calls != 0 {
		t.Fatal("forbidden origin called service")
	}
	w = adminOrderRequest(router, "OPTIONS", "/api/v1/admin/orders/x/status", "", nil, map[string]string{"Origin": "https://shop.example", "Access-Control-Request-Method": "PATCH", "Access-Control-Request-Headers": "Content-Type, Accept"})
	assertAdminHTTP(t, w, 204, "", "")
	if w.Header().Get("Access-Control-Allow-Origin") != "https://shop.example" {
		t.Fatal("preflight origin")
	}
	nilAuth := NewRouter(nil, *logger.NewDiscard(), WithAdminOrders(s, nil, *logger.NewDiscard(), ""))
	assertAdminHTTP(t, adminOrderRequest(nilAuth, "GET", "/api/v1/admin/orders", "", cookie, nil), 403, "ADMIN_REQUIRED", "")
}

func TestAdminOrderRoutesCallService(t *testing.T) {
	router, s, _, cookie := adminOrderUnitRouter(*logger.NewDiscard())
	for _, tc := range []struct{ method, path, body, call, target string }{
		{"GET", "/api/v1/admin/orders/x", "", "get", ""},
		{"PATCH", "/api/v1/admin/orders/x/status", `{"status":"confirmed"}`, "status", "confirmed"},
		{"PATCH", "/api/v1/admin/orders/x/status", `{"status":"shipping"}`, "status", "shipping"},
		{"PATCH", "/api/v1/admin/orders/x/status", `{"status":"delivered"}`, "status", "delivered"},
		{"PATCH", "/api/v1/admin/orders/x/payment", `{"payment_status":"paid"}`, "payment", "paid"},
		{"PATCH", "/api/v1/admin/orders/x/payment", `{"payment_status":"refunded"}`, "payment", "refunded"},
		{"POST", "/api/v1/admin/orders/x/cancel", "", "cancel", ""},
		{"POST", "/api/v1/admin/orders/x/cancel", `{}`, "cancel", ""},
		{"POST", "/api/v1/admin/orders/x/restock", "", "restock", ""},
		{"POST", "/api/v1/admin/orders/x/restock", `{}`, "restock", ""},
	} {
		before := s.calls
		w := adminOrderRequest(router, tc.method, tc.path, tc.body, cookie, map[string]string{"Origin": "https://shop.example"})
		assertAdminHTTP(t, w, 200, "", "")
		if s.calls != before+1 || s.method != tc.call || s.id != "x" || s.target != tc.target {
			t.Fatalf("wrong call: %+v", s)
		}
	}
	for _, tc := range []struct {
		query  string
		filter app.AdminFilter
	}{{"", app.AdminFilter{Page: 1, PageSize: 20}}, {"?status=shipping&payment_status=paid&page=3&page_size=200", app.AdminFilter{Status: "shipping", PaymentStatus: "paid", Page: 3, PageSize: 200}}} {
		w := adminOrderRequest(router, "GET", "/api/v1/admin/orders"+tc.query, "", cookie, nil)
		assertAdminHTTP(t, w, 200, "", "")
		if s.method != "list" || s.filter != tc.filter {
			t.Fatalf("filter: %+v", s.filter)
		}
		var payload struct {
			Data        []adminOrderDTO `json:"data"`
			Total, Page int64
			PageSize    int64 `json:"page_size"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
			t.Fatal(err)
		}
		if len(payload.Data) != 1 || payload.Total != 1 || payload.Page != tc.filter.Page || payload.PageSize != tc.filter.PageSize {
			t.Fatal("list envelope")
		}
	}
	// A bodyless cancel does not require Content-Type.
	assertAdminHTTP(t, adminOrderRequest(router, "POST", "/api/v1/admin/orders/x/cancel", "", cookie, map[string]string{"Content-Type": ""}), 200, "", "")
}

func TestAdminOrderInvalidBodies(t *testing.T) {
	router, s, _, cookie := adminOrderUnitRouter(*logger.NewDiscard())
	for _, tc := range []struct{ name, method, suffix, body, contentType, code, field string }{
		{"unknown", "PATCH", "status", `{"status":"confirmed","extra":true}`, "application/json", "INVALID_BODY", "body"},
		{"malformed", "PATCH", "status", `{"status":`, "application/json", "INVALID_BODY", "body"},
		{"trailing", "PATCH", "status", `{"status":"confirmed"} {}`, "application/json", "INVALID_BODY", "body"},
		{"trailing_garbage", "PATCH", "payment", `{"payment_status":"paid"}x`, "application/json", "INVALID_BODY", "body"},
		{"wrong_type", "PATCH", "status", `{"status":1}`, "application/json", "INVALID_BODY", "body"},
		{"null_object", "PATCH", "status", `null`, "application/json", "INVALID_BODY", "body"},
		{"array", "POST", "cancel", `[]`, "application/json", "INVALID_BODY", "body"},
		{"content_type", "PATCH", "status", `{"status":"confirmed"}`, "text/plain", "INVALID_BODY", "Content-Type"},
		{"fake_json_type", "PATCH", "status", `{"status":"confirmed"}`, "application/json-extra", "INVALID_BODY", "Content-Type"},
		{"oversize", "PATCH", "status", strings.Repeat(" ", 16<<10) + `{}`, "application/json", "INVALID_BODY", "body"},
		{"status_enum", "PATCH", "status", `{"status":"pending"}`, "application/json", "INVALID_FIELD", "status"},
		{"payment_enum", "PATCH", "payment", `{"payment_status":"unpaid"}`, "application/json", "INVALID_FIELD", "payment_status"},
		{"missing_status", "PATCH", "status", `{}`, "application/json", "INVALID_FIELD", "status"},
		{"null_payment", "PATCH", "payment", `{"payment_status":null}`, "application/json", "INVALID_FIELD", "payment_status"},
		{"cancel_field", "POST", "cancel", `{"reason":"x"}`, "application/json", "INVALID_BODY", "body"},
		{"restock_field", "POST", "restock", `{"stock":1}`, "application/json", "INVALID_BODY", "body"},
		{"restock_null", "POST", "restock", `null`, "application/json", "INVALID_BODY", "body"},
		{"cancel_trailing", "POST", "cancel", `{} {}`, "application/json", "INVALID_BODY", "body"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := adminOrderRequest(router, tc.method, "/api/v1/admin/orders/x/"+tc.suffix, tc.body, cookie, map[string]string{"Content-Type": tc.contentType})
			assertAdminHTTP(t, w, 400, tc.code, tc.field)
		})
	}
	if s.calls != 0 {
		t.Fatal("invalid body reached service")
	}
}

func TestAdminOrderListQueryValidation(t *testing.T) {
	router, s, _, cookie := adminOrderUnitRouter(*logger.NewDiscard())
	for _, tc := range []struct{ query, field string }{{"unknown=1", "query"}, {"page=0", "page"}, {"page=-1", "page"}, {"page=100001", "page"}, {"page=1.5", "page"}, {"page=", "page"}, {"page_size=0", "page_size"}, {"page_size=1e2", "page_size"}, {"page_size=9223372036854775808", "page_size"}, {"page=1&page=2", "query"}, {"status=%zz", "query"}} {
		w := adminOrderRequest(router, "GET", "/api/v1/admin/orders?"+tc.query, "", cookie, nil)
		assertAdminHTTP(t, w, 400, "INVALID_FIELD", tc.field)
	}
	if s.calls != 0 {
		t.Fatal("invalid query reached service")
	}
	for _, field := range []string{"status", "payment_status"} {
		s.err = domain.NewInvalidFieldError(field, "Trạng thái không hợp lệ")
		w := adminOrderRequest(router, "GET", "/api/v1/admin/orders?"+field+"=bad", "", cookie, nil)
		assertAdminHTTP(t, w, 400, "INVALID_FIELD", field)
	}
}

func TestAdminOrderErrorMappingAndSafeLogs(t *testing.T) {
	var logs bytes.Buffer
	router, s, _, cookie := adminOrderUnitRouter(zerolog.New(&logs))
	for _, tc := range []struct {
		err         error
		status      int
		code, field string
	}{
		{domain.NewInvalidFieldError("status", "Không hợp lệ"), 400, "INVALID_FIELD", "status"},
		{app.ErrNotFound, 404, "NOT_FOUND", ""},
		{app.ErrStateConflict, 409, "ORDER_STATE_CONFLICT", ""},
		{app.ErrPaymentConflict, 409, "PAYMENT_STATE_CONFLICT", ""},
		{app.ErrRestockUnavailable, 409, "RESTOCK_UNAVAILABLE", ""},
		{app.ErrOrderUnavailable, 503, "ORDER_UNAVAILABLE", ""},
		{errors.New("guest_access_hash SECRET_HASH token SECRET_TOKEN phone 0912345678"), 503, "ORDER_UNAVAILABLE", ""},
	} {
		s.err = fmt.Errorf("wrapped: %w", tc.err)
		w := adminOrderRequest(router, "GET", "/api/v1/admin/orders/x", "", cookie, nil)
		assertAdminHTTP(t, w, tc.status, tc.code, tc.field)
	}
	for _, secret := range []string{"SECRET_HASH", "SECRET_TOKEN", "0912345678"} {
		if strings.Contains(logs.String(), secret) {
			t.Fatal("sensitive error logged")
		}
	}
	if !strings.Contains(logs.String(), "admin order request failed") {
		t.Fatal("unavailable error not logged")
	}
}

func TestAdminOrderDTOAndAllowedActions(t *testing.T) {
	router, s, _, cookie := adminOrderUnitRouter(*logger.NewDiscard())
	for _, tc := range []struct {
		status, method, payment, restock string
		want                             []string
	}{
		{"pending", "cod", "unpaid", "", []string{"confirm", "cancel", "mark_paid"}},
		{"confirmed", "cod", "unpaid", "", []string{"ship", "cancel", "mark_paid"}},
		{"confirmed", "cash_at_counter", "unpaid", "", []string{"cancel", "mark_paid"}},
		{"confirmed", "bank_transfer", "paid", "", []string{"ship", "cancel", "mark_refunded"}},
		{"shipping", "cod", "paid", "", []string{"cancel", "mark_refunded"}},
		{"delivered", "cod", "unpaid", "", []string{"mark_paid"}},
		{"cancelled", "cod", "paid", "awaiting_return", []string{"restock", "mark_refunded"}},
		{"cancelled", "cod", "refunded", "restocked", []string{}},
	} {
		s.order.Status = tc.status
		s.order.PaymentMethod = tc.method
		s.order.PaymentStatus = tc.payment
		s.order.InventoryRestock = tc.restock
		s.order.GuestAccessHash = "SECRET_HASH"
		s.order.Checkout = domain.OrderCheckout{Scope: "SECRET_SCOPE", RequestID: "SECRET_REQUEST", RequestHash: "SECRET_REQUEST_HASH"}
		s.order.Items = []domain.OrderItem{{ProductID: "p", SKU: "sku", Quantity: 1, FulfillmentMode: "ready"}}
		s.order.Note = "Ghi chú"
		w := adminOrderRequest(router, "GET", "/api/v1/admin/orders/x", "", cookie, nil)
		assertAdminHTTP(t, w, 200, "", "")
		var payload struct {
			Data adminOrderDTO `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(payload.Data.AllowedActions, tc.want) {
			t.Fatalf("%s %s %s actions=%v want=%v", tc.status, tc.method, tc.payment, payload.Data.AllowedActions, tc.want)
		}
		if len(payload.Data.Items) != 1 || payload.Data.Items[0].FulfillmentMode != "ready" || !payload.Data.IsGuest || payload.Data.Note != "Ghi chú" {
			t.Fatal("admin DTO fields")
		}
		for _, key := range []string{"guest_access_hash", "checkout", "scope", "request_id", "request_hash", "SECRET_", "customer_id", "cancelled_at", "inventory_restocked_at", "paid_at", "refunded_at"} {
			if strings.Contains(w.Body.String(), key) {
				t.Fatalf("unexpected DTO data: %s", key)
			}
		}
	}
	now := time.Now().UTC().Truncate(time.Millisecond)
	s.order.CustomerID = "customer"
	s.order.CancelledAt = &now
	s.order.InventoryRestockedAt = &now
	s.order.PaidAt = &now
	s.order.RefundedAt = &now
	s.order.UpdatedAt = now
	w := adminOrderRequest(router, "GET", "/api/v1/admin/orders/x", "", cookie, nil)
	assertAdminHTTP(t, w, 200, "", "")
	var payload struct {
		Data adminOrderDTO `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	d := payload.Data
	if d.IsGuest || d.CustomerID != "customer" || d.CancelledAt == nil || d.InventoryRestockedAt == nil || d.PaidAt == nil || d.RefundedAt == nil || !d.UpdatedAt.Equal(now) {
		t.Fatal("timestamps/customer DTO")
	}
}
