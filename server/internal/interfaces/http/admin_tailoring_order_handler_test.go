package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/your-org/aodora/server/internal/application/auth"
	app "github.com/your-org/aodora/server/internal/application/tailoring"
	domainorder "github.com/your-org/aodora/server/internal/domain/order"
	domain "github.com/your-org/aodora/server/internal/domain/tailoring"
	"github.com/your-org/aodora/server/internal/logger"
)

type adminTailoringOrderStub struct {
	calls   int
	id      string
	input   app.CustomOrderInput
	created bool
	err     error
}

func (s *adminTailoringOrderStub) Create(_ context.Context, id string, in app.CustomOrderInput) (*domainorder.Order, bool, error) {
	s.calls++
	s.id, s.input = id, in
	if s.err != nil {
		return nil, false, s.err
	}
	return &domainorder.Order{ID: "7a0000000000000000000001", OrderNumber: "AD-T", FulfillmentMode: "custom", TailoringRequestID: id,
		Status: "pending", PaymentMethod: "cash_at_counter", PaymentStatus: "unpaid"}, s.created, nil
}

func tailoringOrderRouter() (http.Handler, *adminTailoringOrderStub, *authHTTPRepository, *http.Cookie) {
	log := *logger.NewDiscard()
	token := strings.Repeat("c", 64)
	hash := sha256.Sum256([]byte(token))
	repo := &authHTTPRepository{user: auth.User{ID: "680000000000000000000001", Role: "admin"}, tokenHash: hex.EncodeToString(hash[:]), expiresAt: time.Now().Add(time.Hour)}
	guard := NewAuthGuard(auth.NewService(repo, time.Hour), AuthCookieConfig{Name: "session"})
	stub := &adminTailoringOrderStub{created: true}
	router := NewRouter(nil, log, WithAdminCatalog(nil, guard.RequireAdmin, log, "https://shop.example"),
		WithAdminTailoringOrders(stub, guard.RequireAdmin, log, "https://shop.example"))
	return router, stub, repo, &http.Cookie{Name: "session", Value: token}
}

const customOrderPath = "/api/v1/admin/tailoring-requests/6d0000000000000000000001/order"

func TestAdminTailoringOrderAuthorization(t *testing.T) {
	router, s, repo, cookie := tailoringOrderRouter()
	body := `{"unit_price":1000,"payment_method":"cash_at_counter"}`
	for _, role := range []string{"guest", "customer"} {
		repo.user.Role = role
		c := cookie
		if role == "guest" {
			c = nil
		}
		assertAdminHTTP(t, adminOrderRequest(router, "POST", customOrderPath, body, c, nil), 403, "ADMIN_REQUIRED", "")
	}
	repo.user.Role = "admin"
	assertAdminHTTP(t, adminOrderRequest(router, "POST", customOrderPath, body, cookie, map[string]string{"Origin": "https://evil.example"}), 403, "ORIGIN_FORBIDDEN", "")
	if s.calls != 0 {
		t.Fatal("forbidden request reached service")
	}
}

func TestAdminTailoringOrderBody(t *testing.T) {
	router, s, _, cookie := tailoringOrderRouter()
	for name, c := range map[string]struct{ body, code, field string }{
		"no price":        {`{"payment_method":"cod"}`, "INVALID_FIELD", "unit_price"},
		"null price":      {`{"unit_price":null,"payment_method":"cod"}`, "INVALID_FIELD", "unit_price"},
		"no method":       {`{"unit_price":1}`, "INVALID_FIELD", "payment_method"},
		"unknown field":   {`{"unit_price":1,"payment_method":"cod","total":5}`, "INVALID_BODY", "body"},
		"customer_id":     {`{"unit_price":1,"payment_method":"cod","customer_id":"x"}`, "INVALID_BODY", "body"},
		"status":          {`{"unit_price":1,"payment_method":"cod","status":"paid"}`, "INVALID_BODY", "body"},
		"float quantity":  {`{"unit_price":1,"payment_method":"cod","quantity":1.5}`, "INVALID_BODY", "body"},
		"string quantity": {`{"unit_price":1,"payment_method":"cod","quantity":"2"}`, "INVALID_BODY", "body"},
		"fee field":       {`{"unit_price":1,"payment_method":"cod","shipping_fee":0}`, "INVALID_BODY", "body"},
	} {
		w := adminOrderRequest(router, "POST", customOrderPath, c.body, cookie, nil)
		assertAdminHTTP(t, w, 400, c.code, c.field)
		_ = name
	}
	if s.calls != 0 {
		t.Fatal("invalid body reached service")
	}
	// exact price literal and string price pass through untouched
	for body, want := range map[string]string{
		`{"unit_price":1500000.50,"payment_method":"cash_at_counter"}`:  "1500000.50",
		`{"unit_price":"2500000","payment_method":"cash_at_counter"}`:   "2500000",
		`{"unit_price":0.1,"payment_method":"cash_at_counter"}`:         "0.1",
		`{"unit_price":12345678901234567890123,"payment_method":"cod"}`: "12345678901234567890123",
	} {
		assertAdminHTTP(t, adminOrderRequest(router, "POST", customOrderPath, body, cookie, nil), 201, "", "")
		if s.input.UnitPrice != want {
			t.Fatalf("unit_price %q want %q", s.input.UnitPrice, want)
		}
	}
}

func TestAdminTailoringOrderResponseAndErrors(t *testing.T) {
	router, s, _, cookie := tailoringOrderRouter()
	body := `{"product_id":"680000000000000000000001","unit_price":1000,"quantity":2,"payment_method":"cod","shipping_address":{"line1":"1 A","city":"Huế"},"note":"gấp"}`
	w := adminOrderRequest(router, "POST", customOrderPath, body, cookie, nil)
	assertAdminHTTP(t, w, 201, "", "")
	if s.id != "6d0000000000000000000001" || *s.input.ProductID != "680000000000000000000001" || *s.input.Quantity != 2 ||
		s.input.Address.Line1 != "1 A" || s.input.Address.City != "Huế" || s.input.Note != "gấp" {
		t.Fatalf("input: %+v", s.input)
	}
	if !strings.Contains(w.Body.String(), `"fulfillment_mode":"custom"`) || !strings.Contains(w.Body.String(), `"tailoring_request_id":"6d0000000000000000000001"`) || !strings.Contains(w.Body.String(), `"created":true`) {
		t.Fatalf("body: %s", w.Body.String())
	}
	s.created = false
	assertAdminHTTP(t, adminOrderRequest(router, "POST", customOrderPath, body, cookie, nil), 200, "", "")
	for _, c := range []struct {
		err         error
		status      int
		code, field string
	}{
		{app.ErrNotFound, 404, "NOT_FOUND", ""},
		{app.ErrStateConflict, 409, "TAILORING_STATE_CONFLICT", ""},
		{app.ErrMeasurementIncomplete, 409, "MEASUREMENT_INCOMPLETE", "measurements"},
		{app.ErrCustomOrderExists, 409, "CUSTOM_ORDER_EXISTS", ""},
		{app.ErrShippingUnavailable, 503, "SHIPPING_UNAVAILABLE", ""},
		{domain.NewInvalidFieldError("unit_price", "x"), 400, "INVALID_FIELD", "unit_price"},
		{domain.ErrTailoringUnavailable, 503, "TAILORING_UNAVAILABLE", ""},
		{errors.New("secret 0900000001"), 503, "TAILORING_UNAVAILABLE", ""},
	} {
		s.err = c.err
		w := adminOrderRequest(router, "POST", customOrderPath, body, cookie, nil)
		assertAdminHTTP(t, w, c.status, c.code, c.field)
		if strings.Contains(w.Body.String(), "secret") || strings.Contains(w.Body.String(), "0900000001") {
			t.Fatalf("leak: %s", w.Body.String())
		}
	}
}
