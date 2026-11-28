package httpapi

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
	appauth "github.com/your-org/aodora/server/internal/application/auth"
	apporder "github.com/your-org/aodora/server/internal/application/order"
	domainorder "github.com/your-org/aodora/server/internal/domain/order"
	"github.com/your-org/aodora/server/internal/logger"
)

type stubOrderService struct {
	quoteFunc    func(ctx context.Context, input domainorder.CheckoutInput) (*apporder.QuoteResult, error)
	createFunc   func(ctx context.Context, principal apporder.Principal, requestID string, input domainorder.CheckoutInput) (domainorder.Order, bool, error)
	getFunc      func(ctx context.Context, viewer apporder.Viewer, id string) (*domainorder.Order, error)
	listFunc     func(ctx context.Context, customerID, status string, page, pageSize int64) ([]domainorder.Order, int64, int64, int64, error)
	attemptFunc  func(ctx context.Context, viewer apporder.Viewer, requestID string) (*domainorder.Order, error)
	receivedFunc func(ctx context.Context, viewer apporder.Viewer, id string) (*domainorder.Order, error)

	lastPrincipal apporder.Principal
	lastViewer    apporder.Viewer
	lastRequestID string
	lastInput     domainorder.CheckoutInput
	called        bool
}

func (s *stubOrderService) Quote(ctx context.Context, input domainorder.CheckoutInput) (*apporder.QuoteResult, error) {
	s.called = true
	s.lastInput = input
	if s.quoteFunc != nil {
		return s.quoteFunc(ctx, input)
	}
	return nil, nil
}

func (s *stubOrderService) Create(ctx context.Context, principal apporder.Principal, requestID string, input domainorder.CheckoutInput) (domainorder.Order, bool, error) {
	s.called = true
	s.lastPrincipal = principal
	s.lastRequestID = requestID
	s.lastInput = input
	if s.createFunc != nil {
		return s.createFunc(ctx, principal, requestID, input)
	}
	return domainorder.Order{}, false, nil
}

func (s *stubOrderService) ConfirmReceived(ctx context.Context, viewer apporder.Viewer, id string) (*domainorder.Order, error) {
	s.called = true
	s.lastViewer = viewer
	if s.receivedFunc != nil {
		return s.receivedFunc(ctx, viewer, id)
	}
	return nil, apporder.ErrNotFound
}

func (s *stubOrderService) Get(ctx context.Context, viewer apporder.Viewer, id string) (*domainorder.Order, error) {
	s.called = true
	s.lastViewer = viewer
	if s.getFunc != nil {
		return s.getFunc(ctx, viewer, id)
	}
	return nil, nil
}

func (s *stubOrderService) List(ctx context.Context, customerID, status string, page, pageSize int64) ([]domainorder.Order, int64, int64, int64, error) {
	s.called = true
	if s.listFunc != nil {
		return s.listFunc(ctx, customerID, status, page, pageSize)
	}
	return nil, 0, page, pageSize, nil
}

func (s *stubOrderService) Attempt(ctx context.Context, viewer apporder.Viewer, requestID string) (*domainorder.Order, error) {
	s.called = true
	s.lastViewer = viewer
	s.lastRequestID = requestID
	if s.attemptFunc != nil {
		return s.attemptFunc(ctx, viewer, requestID)
	}
	return nil, nil
}

func setupOrderTestRouter(service OrderService, allowedOrigin string) (*http.ServeMux, *AuthGuard, *authHTTPRepository, *appauth.Service) {
	repo := &authHTTPRepository{}
	authService := appauth.NewService(repo, time.Hour)
	guard := NewAuthGuard(authService, AuthCookieConfig{
		Name:          "session",
		TTL:           time.Hour,
		AllowedOrigin: allowedOrigin,
	})
	mux := http.NewServeMux()
	WithAuth(guard)(mux)
	WithOrders(service, guard, *logger.NewDiscard())(mux)
	return mux, guard, repo, authService
}

func createGuestTokenFixture() (rawToken string, hashHex string) {
	bytes32 := make([]byte, 32)
	for i := 0; i < 32; i++ {
		bytes32[i] = byte(i)
	}
	rawToken = base64.RawURLEncoding.EncodeToString(bytes32)
	h := sha256.Sum256(bytes32)
	hashHex = hex.EncodeToString(h[:])
	return rawToken, hashHex
}

func sampleOrder(id, orderNumber string) domainorder.Order {
	p500k, _ := domainorder.ParseMoney("500000")
	p30k, _ := domainorder.ParseMoney("30000")
	p530k, _ := domainorder.ParseMoney("530000")
	return domainorder.Order{
		ID:            id,
		OrderNumber:   orderNumber,
		Status:        domainorder.StatusPending,
		PaymentMethod: "cod",
		PaymentStatus: domainorder.PaymentUnpaid,
		Subtotal:      p500k,
		ShippingFee:   p30k,
		Total:         p530k,
		ShippingAddress: domainorder.NormalizedAddress{
			Recipient: "Nguyen An",
			Phone:     "0900000001",
			Line1:     "12 Nguyen Trai",
			City:      "TP. Ho Chi Minh",
		},
		Items: []domainorder.OrderItem{
			{
				ProductID:   "680000000000000000000001",
				ProductName: "Ao Dai Lua",
				SKU:         "LUA-DO-M",
				Quantity:    1,
				UnitPrice:   p500k,
			},
		},
		Note:      "Giao gio hanh chinh",
		CreatedAt: time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC),
	}
}

func TestQuoteSuccess200(t *testing.T) {
	subtotal, _ := domainorder.ParseMoney("500000")
	fee, _ := domainorder.ParseMoney("30000")
	total, _ := domainorder.ParseMoney("530000")
	unitPrice, _ := domainorder.ParseMoney("500000")
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)

	service := &stubOrderService{
		quoteFunc: func(_ context.Context, _ domainorder.CheckoutInput) (*apporder.QuoteResult, error) {
			return &apporder.QuoteResult{
				QuoteToken:  "test-quote-token-opaque",
				ExpiresAt:   now.Add(15 * time.Minute),
				Subtotal:    subtotal,
				ShippingFee: fee,
				Total:       total,
				Items: []apporder.QuoteItemResult{
					{
						ProductID: "680000000000000000000001",
						SKU:       "LUA-DO-M",
						Quantity:  1,
						UnitPrice: unitPrice,
					},
				},
			}, nil
		},
	}
	router, _, _, _ := setupOrderTestRouter(service, "https://shop.example")

	body := `{
		"payment_method": "cod",
		"shipping_address": {
			"recipient": "Nguyen An",
			"phone": "0900000001",
			"line1": "12 Nguyen Trai",
			"city": "TP. Ho Chi Minh"
		},
		"items": [
			{"product_id": "680000000000000000000001", "sku": "LUA-DO-M", "quantity": 1}
		]
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/orders/quote", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
	if w.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("expected Cache-Control: no-store, got %s", w.Header().Get("Cache-Control"))
	}

	var res struct {
		Data struct {
			QuoteToken  string `json:"quote_token"`
			ExpiresAt   string `json:"expires_at"`
			Subtotal    int64  `json:"subtotal"`
			ShippingFee int64  `json:"shipping_fee"`
			Total       int64  `json:"total"`
			Items       []struct {
				ProductID string `json:"product_id"`
				SKU       string `json:"sku"`
				Quantity  int64  `json:"quantity"`
				UnitPrice int64  `json:"unit_price"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to parse quote response: %v", err)
	}
	if res.Data.QuoteToken != "test-quote-token-opaque" || res.Data.Subtotal != 500000 || res.Data.Total != 530000 {
		t.Fatalf("unexpected quote response data: %+v", res.Data)
	}
	// Verify raw JSON serializes money as numbers, not strings
	raw := w.Body.String()
	if !strings.Contains(raw, `"subtotal":500000`) || !strings.Contains(raw, `"total":530000`) {
		t.Fatalf("expected money serialized as raw number: %s", raw)
	}
}

func TestCreateSuccess201AndReplay200(t *testing.T) {
	token, _ := createGuestTokenFixture()
	orderSample := sampleOrder("690000000000000000000010", "AD-261116-TEST0001")

	var isCreated bool
	service := &stubOrderService{
		createFunc: func(_ context.Context, principal apporder.Principal, _ string, _ domainorder.CheckoutInput) (domainorder.Order, bool, error) {
			if principal.GuestHash == "" {
				return domainorder.Order{}, false, errors.New("missing guest hash")
			}
			return orderSample, isCreated, nil
		},
	}
	router, _, _, _ := setupOrderTestRouter(service, "https://shop.example")

	body := `{
		"payment_method": "cod",
		"shipping_address": {
			"recipient": "Nguyen An",
			"phone": "0900000001",
			"line1": "12 Nguyen Trai",
			"city": "TP. Ho Chi Minh"
		},
		"items": [
			{"product_id": "680000000000000000000001", "sku": "LUA-DO-M", "quantity": 1}
		],
		"quote_token": "valid-quote-token"
	}`

	// 1. Initial creation (201)
	isCreated = true
	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/orders", strings.NewReader(body))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Idempotency-Key", "ad577a65-cd42-46eb-ae8a-98b5f1186571")
	req1.Header.Set("X-Order-Access-Token", token)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	if w1.Code != http.StatusCreated {
		t.Fatalf("expected status 201 on new creation, got %d: %s", w1.Code, w1.Body.String())
	}
	if w1.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("expected Cache-Control: no-store, got %s", w1.Header().Get("Cache-Control"))
	}
	respStr := w1.Body.String()
	for _, forbidden := range []string{"guest_access_hash", "checkout", "request_hash", "scope", "customer_id", token} {
		if strings.Contains(respStr, forbidden) {
			t.Fatalf("response contains forbidden field/secret %q: %s", forbidden, respStr)
		}
	}

	// 2. Replay with same key & payload (200)
	isCreated = false
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/orders", strings.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Idempotency-Key", "ad577a65-cd42-46eb-ae8a-98b5f1186571")
	req2.Header.Set("X-Order-Access-Token", token)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("expected status 200 on replay, got %d: %s", w2.Code, w2.Body.String())
	}
}

func TestNormalizePaymentMethodBankTransferCardEWallet(t *testing.T) {
	token, _ := createGuestTokenFixture()
	service := &stubOrderService{
		createFunc: func(_ context.Context, _ apporder.Principal, _ string, input domainorder.CheckoutInput) (domainorder.Order, bool, error) {
			_, err := domainorder.Normalize(input)
			if err != nil {
				return domainorder.Order{}, false, err
			}
			return domainorder.Order{}, true, nil
		},
	}
	router, _, _, _ := setupOrderTestRouter(service, "https://shop.example")

	for _, pm := range []string{"bank_transfer", "card", "e_wallet", "unknown"} {
		body := fmt.Sprintf(`{
			"payment_method": %q,
			"shipping_address": {"recipient": "Nguyen An", "phone": "0900000001"},
			"items": [{"product_id": "680000000000000000000001", "sku": "LUA-DO-M", "quantity": 1}],
			"quote_token": "token"
		}`, pm)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/orders", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", "ad577a65-cd42-46eb-ae8a-98b5f1186571")
		req.Header.Set("X-Order-Access-Token", token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for payment_method=%s, got %d: %s", pm, w.Code, w.Body.String())
		}
		var errResp map[string]apiError
		if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
			t.Fatal(err)
		}
		if errResp["error"].Code != "INVALID_FIELD" || errResp["error"].Field != "payment_method" {
			t.Fatalf("expected INVALID_FIELD on payment_method, got %+v", errResp)
		}
	}
}

func TestInvalidBodyScenarios(t *testing.T) {
	token, _ := createGuestTokenFixture()
	service := &stubOrderService{}
	router, _, _, _ := setupOrderTestRouter(service, "https://shop.example")

	testCases := []struct {
		name        string
		url         string
		contentType string
		body        string
	}{
		{
			name:        "customer_id in body",
			url:         "/api/v1/orders",
			contentType: "application/json",
			body:        `{"customer_id":"670000000000000000000001","payment_method":"cod","shipping_address":{"recipient":"A","phone":"090"},"items":[{"product_id":"680000000000000000000001","sku":"S","quantity":1}],"quote_token":"t"}`,
		},
		{
			name:        "unit_price in body",
			url:         "/api/v1/orders",
			contentType: "application/json",
			body:        `{"payment_method":"cod","shipping_address":{"recipient":"A","phone":"090"},"items":[{"product_id":"680000000000000000000001","sku":"S","quantity":1,"unit_price":500000}],"quote_token":"t"}`,
		},
		{
			name:        "status in body",
			url:         "/api/v1/orders",
			contentType: "application/json",
			body:        `{"status":"pending","payment_method":"cod","shipping_address":{"recipient":"A","phone":"090"},"items":[{"product_id":"680000000000000000000001","sku":"S","quantity":1}],"quote_token":"t"}`,
		},
		{
			name:        "total in body",
			url:         "/api/v1/orders",
			contentType: "application/json",
			body:        `{"total":500000,"payment_method":"cod","shipping_address":{"recipient":"A","phone":"090"},"items":[{"product_id":"680000000000000000000001","sku":"S","quantity":1}],"quote_token":"t"}`,
		},
		{
			name:        "quote_token in quote request",
			url:         "/api/v1/orders/quote",
			contentType: "application/json",
			body:        `{"quote_token":"surprise","payment_method":"cod","shipping_address":{"recipient":"A","phone":"090"},"items":[{"product_id":"680000000000000000000001","sku":"S","quantity":1}]}`,
		},
		{
			name:        "malformed JSON",
			url:         "/api/v1/orders",
			contentType: "application/json",
			body:        `{"payment_method": "cod",`,
		},
		{
			name:        "trailing data after JSON object",
			url:         "/api/v1/orders",
			contentType: "application/json",
			body:        `{"payment_method":"cod","shipping_address":{"recipient":"A","phone":"090"},"items":[{"product_id":"680000000000000000000001","sku":"S","quantity":1}],"quote_token":"t"} extra_garbage`,
		},
		{
			name:        "wrong Content-Type text/plain",
			url:         "/api/v1/orders",
			contentType: "text/plain",
			body:        `{"payment_method":"cod"}`,
		},
		{
			name:        "body > 64 KiB",
			url:         "/api/v1/orders",
			contentType: "application/json",
			body:        `{"payment_method":"cod","note":"` + strings.Repeat("A", 65*1024) + `"}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			service.called = false
			req := httptest.NewRequest(http.MethodPost, tc.url, strings.NewReader(tc.body))
			req.Header.Set("Content-Type", tc.contentType)
			req.Header.Set("Idempotency-Key", "ad577a65-cd42-46eb-ae8a-98b5f1186571")
			req.Header.Set("X-Order-Access-Token", token)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Fatalf("%s: expected 400, got %d: %s", tc.name, w.Code, w.Body.String())
			}
			var errResp map[string]apiError
			if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
				t.Fatal(err)
			}
			if errResp["error"].Code != "INVALID_BODY" {
				t.Fatalf("%s: expected INVALID_BODY, got %s", tc.name, errResp["error"].Code)
			}
			if service.called {
				t.Fatalf("%s: service should NOT have been called", tc.name)
			}
		})
	}
}

func TestInvalidQuantityScenarios(t *testing.T) {
	token, _ := createGuestTokenFixture()
	service := &stubOrderService{}
	router, _, _, _ := setupOrderTestRouter(service, "https://shop.example")

	invalidQuantities := []string{
		`1.5`,
		`"1"`,
		`1e2`,
		`0`,
		`-1`,
	}

	for _, q := range invalidQuantities {
		t.Run("quantity_"+q, func(t *testing.T) {
			service.called = false
			body := fmt.Sprintf(`{
				"payment_method": "cash_at_counter",
				"shipping_address": {"recipient": "A", "phone": "0900000001"},
				"items": [{"product_id": "680000000000000000000001", "sku": "SKU-1", "quantity": %s}],
				"quote_token": "token"
			}`, q)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/orders", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Idempotency-Key", "ad577a65-cd42-46eb-ae8a-98b5f1186571")
			req.Header.Set("X-Order-Access-Token", token)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Fatalf("quantity %s: expected 400, got %d: %s", q, w.Code, w.Body.String())
			}
			var errResp map[string]apiError
			if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
				t.Fatal(err)
			}
			if errResp["error"].Code != "INVALID_FIELD" || errResp["error"].Field != "items[0].quantity" {
				t.Fatalf("quantity %s: expected INVALID_FIELD items[0].quantity, got %+v", q, errResp)
			}
			if service.called {
				t.Fatalf("quantity %s: service should not have been called", q)
			}
		})
	}
}

func TestAuthCookieInvalidRejection(t *testing.T) {
	service := &stubOrderService{}
	router, _, _, _ := setupOrderTestRouter(service, "https://shop.example")

	routes := []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodPost, "/api/v1/orders/quote", `{"payment_method":"cod"}`},
		{http.MethodPost, "/api/v1/orders", `{"payment_method":"cod"}`},
		{http.MethodGet, "/api/v1/orders/690000000000000000000010", ""},
		{http.MethodGet, "/api/v1/orders/attempts/ad577a65-cd42-46eb-ae8a-98b5f1186571", ""},
		{http.MethodGet, "/api/v1/orders", ""},
	}

	for _, r := range routes {
		t.Run(r.method+" "+r.path, func(t *testing.T) {
			service.called = false
			var reader *strings.Reader
			if r.body != "" {
				reader = strings.NewReader(r.body)
			} else {
				reader = strings.NewReader("")
			}
			req := httptest.NewRequest(r.method, r.path, reader)
			if r.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			// Cookie with invalid session token
			req.AddCookie(&http.Cookie{Name: "session", Value: "invalid-token-xyz"})
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusUnauthorized {
				t.Fatalf("%s %s: expected 401 AUTH_REQUIRED, got %d: %s", r.method, r.path, w.Code, w.Body.String())
			}
			var errResp map[string]apiError
			if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
				t.Fatal(err)
			}
			if errResp["error"].Code != "AUTH_REQUIRED" {
				t.Fatalf("expected AUTH_REQUIRED, got %s", errResp["error"].Code)
			}
			if service.called {
				t.Fatalf("%s %s: service should NOT have been called", r.method, r.path)
			}
		})
	}
}

func TestGuestTokenValidationAndPOSTCredentials(t *testing.T) {
	validToken, _ := createGuestTokenFixture()
	service := &stubOrderService{}
	router, _, _, authService := setupOrderTestRouter(service, "https://shop.example")
	if _, err := authService.Register(context.Background(), appauth.RegisterInput{FullName: "Customer", Email: "c@example.com", Password: "password-123"}); err != nil {
		t.Fatal(err)
	}
	_, custSession, _, err := authService.Login(context.Background(), "c@example.com", "password-123")
	if err != nil {
		t.Fatal(err)
	}

	// 1. Non-canonical or malformed guest token -> 400
	for _, badToken := range []string{"too-short", strings.Repeat("A", 44), "====", validToken[:42] + "="} {
		service.called = false
		req := httptest.NewRequest(http.MethodGet, "/api/v1/orders/690000000000000000000010", nil)
		req.Header.Set("X-Order-Access-Token", badToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("bad token %q: expected 400, got %d: %s", badToken, w.Code, w.Body.String())
		}
		var errResp map[string]apiError
		_ = json.Unmarshal(w.Body.Bytes(), &errResp)
		if errResp["error"].Code != "INVALID_FIELD" || errResp["error"].Field != "X-Order-Access-Token" {
			t.Fatalf("expected INVALID_FIELD X-Order-Access-Token, got %+v", errResp)
		}
		if service.called {
			t.Fatalf("bad token: service should not have been called")
		}
	}

	// 2. Both valid session AND guest token at POST /api/v1/orders -> 400
	service.called = false
	body := `{"payment_method":"cash_at_counter","shipping_address":{"recipient":"A","phone":"090"},"items":[{"product_id":"680000000000000000000001","sku":"S","quantity":1}],"quote_token":"t"}`
	reqBoth := httptest.NewRequest(http.MethodPost, "/api/v1/orders", strings.NewReader(body))
	reqBoth.Header.Set("Content-Type", "application/json")
	reqBoth.Header.Set("Idempotency-Key", "ad577a65-cd42-46eb-ae8a-98b5f1186571")
	reqBoth.Header.Set("X-Order-Access-Token", validToken)
	reqBoth.AddCookie(&http.Cookie{Name: "session", Value: custSession})
	wBoth := httptest.NewRecorder()
	router.ServeHTTP(wBoth, reqBoth)

	if wBoth.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 when sending both session and token, got %d: %s", wBoth.Code, wBoth.Body.String())
	}
	if service.called {
		t.Fatalf("service should not have been called when both credentials sent")
	}

	// 3. Neither session NOR guest token at POST /api/v1/orders -> 400
	service.called = false
	reqNeither := httptest.NewRequest(http.MethodPost, "/api/v1/orders", strings.NewReader(body))
	reqNeither.Header.Set("Content-Type", "application/json")
	reqNeither.Header.Set("Idempotency-Key", "ad577a65-cd42-46eb-ae8a-98b5f1186571")
	wNeither := httptest.NewRecorder()
	router.ServeHTTP(wNeither, reqNeither)

	if wNeither.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 when sending no credentials, got %d: %s", wNeither.Code, wNeither.Body.String())
	}

	// 4. Missing or invalid Idempotency-Key at POST /api/v1/orders -> 400
	for _, badKey := range []string{"", "not-a-uuid", "12345"} {
		service.called = false
		reqKey := httptest.NewRequest(http.MethodPost, "/api/v1/orders", strings.NewReader(body))
		reqKey.Header.Set("Content-Type", "application/json")
		if badKey != "" {
			reqKey.Header.Set("Idempotency-Key", badKey)
		}
		reqKey.Header.Set("X-Order-Access-Token", validToken)
		wKey := httptest.NewRecorder()
		router.ServeHTTP(wKey, reqKey)

		if wKey.Code != http.StatusBadRequest {
			t.Fatalf("bad key %q: expected 400, got %d: %s", badKey, wKey.Code, wKey.Body.String())
		}
		var errResp map[string]apiError
		_ = json.Unmarshal(wKey.Body.Bytes(), &errResp)
		if errResp["error"].Code != "INVALID_FIELD" || errResp["error"].Field != "Idempotency-Key" {
			t.Fatalf("expected INVALID_FIELD Idempotency-Key, got %+v", errResp)
		}
	}
}

func TestViewerAuthorizationAndQueryConstraints(t *testing.T) {
	validToken, fixtureHash := createGuestTokenFixture()
	orderSample := sampleOrder("690000000000000000000010", "AD-261116-TEST0001")

	var lastStatus string
	service := &stubOrderService{
		getFunc: func(_ context.Context, viewer apporder.Viewer, id string) (*domainorder.Order, error) {
			if id == orderSample.ID {
				return &orderSample, nil
			}
			return nil, apporder.ErrNotFound
		},
		attemptFunc: func(_ context.Context, viewer apporder.Viewer, requestID string) (*domainorder.Order, error) {
			return &orderSample, nil
		},
		listFunc: func(_ context.Context, customerID, status string, page, pageSize int64) ([]domainorder.Order, int64, int64, int64, error) {
			lastStatus = status
			return []domainorder.Order{orderSample}, 1, page, pageSize, nil
		},
	}
	router, _, authRepo, authService := setupOrderTestRouter(service, "https://shop.example")
	if _, err := authService.Register(context.Background(), appauth.RegisterInput{FullName: "Admin", Email: "admin@example.com", Password: "password-123"}); err != nil {
		t.Fatal(err)
	}
	_, adminSession, _, err := authService.Login(context.Background(), "admin@example.com", "password-123")
	if err != nil {
		t.Fatal(err)
	}
	authRepo.user.Role = "admin"
	authRepo.user.ID = "670000000000000000000099"

	// 1. GET {id} and attempts without credential -> 404 NOT_FOUND
	service.called = false
	reqGetNoCred := httptest.NewRequest(http.MethodGet, "/api/v1/orders/690000000000000000000010", nil)
	wGetNoCred := httptest.NewRecorder()
	router.ServeHTTP(wGetNoCred, reqGetNoCred)
	if wGetNoCred.Code != http.StatusNotFound || service.called {
		t.Fatalf("expected 404 for GET without credential, got %d (service called=%v)", wGetNoCred.Code, service.called)
	}

	service.called = false
	reqAttNoCred := httptest.NewRequest(http.MethodGet, "/api/v1/orders/attempts/ad577a65-cd42-46eb-ae8a-98b5f1186571", nil)
	wAttNoCred := httptest.NewRecorder()
	router.ServeHTTP(wAttNoCred, reqAttNoCred)
	if wAttNoCred.Code != http.StatusNotFound || service.called {
		t.Fatalf("expected 404 for attempt without credential, got %d (service called=%v)", wAttNoCred.Code, service.called)
	}

	// 2. GET list without session -> 401 AUTH_REQUIRED
	service.called = false
	reqListNoSession := httptest.NewRequest(http.MethodGet, "/api/v1/orders", nil)
	reqListNoSession.Header.Set("X-Order-Access-Token", validToken) // guest token cannot replace session
	wListNoSession := httptest.NewRecorder()
	router.ServeHTTP(wListNoSession, reqListNoSession)
	if wListNoSession.Code != http.StatusUnauthorized || service.called {
		t.Fatalf("expected 401 for GET list with guest token, got %d", wListNoSession.Code)
	}

	// 3. Guest GET {id} passes correct GuestHash fixture to Viewer
	service.called = false
	reqGuest := httptest.NewRequest(http.MethodGet, "/api/v1/orders/690000000000000000000010", nil)
	reqGuest.Header.Set("X-Order-Access-Token", validToken)
	wGuest := httptest.NewRecorder()
	router.ServeHTTP(wGuest, reqGuest)
	if wGuest.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", wGuest.Code)
	}
	if service.lastViewer.GuestHash != fixtureHash {
		t.Fatalf("expected Viewer.GuestHash=%s, got %s", fixtureHash, service.lastViewer.GuestHash)
	}

	// 4. Admin session sets Viewer.IsAdmin = true
	service.called = false
	reqAdmin := httptest.NewRequest(http.MethodGet, "/api/v1/orders/690000000000000000000010", nil)
	reqAdmin.AddCookie(&http.Cookie{Name: "session", Value: adminSession})
	wAdmin := httptest.NewRecorder()
	router.ServeHTTP(wAdmin, reqAdmin)
	if wAdmin.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", wAdmin.Code)
	}
	if !service.lastViewer.IsAdmin || service.lastViewer.CustomerID != "670000000000000000000099" {
		t.Fatalf("expected Viewer.IsAdmin=true, got %+v", service.lastViewer)
	}

	// 5. Query validation for list: page, page_size, query parameter check
	badQueries := []string{
		"?unknown=1",
		"?page=0",
		"?page=100001",
		"?page=abc",
		"?page_size=0",
		"?page_size=-5",
		"?page_size=xyz",
		"?status=pending&status=delivered",
	}
	for _, bq := range badQueries {
		reqBQ := httptest.NewRequest(http.MethodGet, "/api/v1/orders"+bq, nil)
		reqBQ.AddCookie(&http.Cookie{Name: "session", Value: adminSession})
		wBQ := httptest.NewRecorder()
		router.ServeHTTP(wBQ, reqBQ)
		if wBQ.Code != http.StatusBadRequest {
			t.Fatalf("query %s: expected 400, got %d", bq, wBQ.Code)
		}
	}

	// 6. status is passed through to the service (which validates the value).
	reqStatus := httptest.NewRequest(http.MethodGet, "/api/v1/orders?status=delivered&page=1", nil)
	reqStatus.AddCookie(&http.Cookie{Name: "session", Value: adminSession})
	wStatus := httptest.NewRecorder()
	router.ServeHTTP(wStatus, reqStatus)
	if wStatus.Code != http.StatusOK || lastStatus != "delivered" {
		t.Fatalf("status filter: code=%d status=%q", wStatus.Code, lastStatus)
	}
}

func TestErrorMapping(t *testing.T) {
	token, _ := createGuestTokenFixture()

	testErrors := []struct {
		err            error
		expectedStatus int
		expectedCode   string
	}{
		{domainorder.NewInvalidFieldError("test_field", "field is invalid"), http.StatusBadRequest, "INVALID_FIELD"},
		{apporder.ErrNotFound, http.StatusNotFound, "NOT_FOUND"},
		{apporder.ErrIdempotencyConflict, http.StatusConflict, "IDEMPOTENCY_CONFLICT"},
		{apporder.ErrOutOfStock, http.StatusConflict, "OUT_OF_STOCK"},
		{apporder.ErrQuoteChanged, http.StatusConflict, "QUOTE_CHANGED"},
		{apporder.ErrQuoteExpired, http.StatusConflict, "QUOTE_EXPIRED"},
		{apporder.ErrShippingUnavailable, http.StatusServiceUnavailable, "SHIPPING_UNAVAILABLE"},
		{apporder.ErrOrderUnavailable, http.StatusServiceUnavailable, "ORDER_UNAVAILABLE"},
		{errors.New("raw unhandled DB error"), http.StatusServiceUnavailable, "ORDER_UNAVAILABLE"},
	}

	for _, tc := range testErrors {
		t.Run(tc.expectedCode, func(t *testing.T) {
			service := &stubOrderService{
				createFunc: func(context.Context, apporder.Principal, string, domainorder.CheckoutInput) (domainorder.Order, bool, error) {
					return domainorder.Order{}, false, tc.err
				},
			}
			router, _, _, _ := setupOrderTestRouter(service, "https://shop.example")

			body := `{"payment_method":"cash_at_counter","shipping_address":{"recipient":"A","phone":"090"},"items":[{"product_id":"680000000000000000000001","sku":"S","quantity":1}],"quote_token":"t"}`
			req := httptest.NewRequest(http.MethodPost, "/api/v1/orders", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Idempotency-Key", "ad577a65-cd42-46eb-ae8a-98b5f1186571")
			req.Header.Set("X-Order-Access-Token", token)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tc.expectedStatus {
				t.Fatalf("err %v: expected status %d, got %d: %s", tc.err, tc.expectedStatus, w.Code, w.Body.String())
			}
			var errResp map[string]apiError
			if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
				t.Fatal(err)
			}
			if errResp["error"].Code != tc.expectedCode {
				t.Fatalf("expected code %s, got %s", tc.expectedCode, errResp["error"].Code)
			}
			if w.Header().Get("Cache-Control") != "no-store" {
				t.Fatalf("expected Cache-Control: no-store, got %s", w.Header().Get("Cache-Control"))
			}
		})
	}
}

func TestCORSAndPreflight(t *testing.T) {
	service := &stubOrderService{}
	router, _, _, _ := setupOrderTestRouter(service, "https://shop.example")

	// 1. Allowed origin preflight with all 3 headers -> 204 No Content
	reqOptions := httptest.NewRequest(http.MethodOptions, "/api/v1/orders", nil)
	reqOptions.Header.Set("Origin", "https://shop.example")
	reqOptions.Header.Set("Access-Control-Request-Method", http.MethodPost)
	reqOptions.Header.Set("Access-Control-Request-Headers", "content-type, idempotency-key, x-order-access-token")
	wOptions := httptest.NewRecorder()
	router.ServeHTTP(wOptions, reqOptions)

	if wOptions.Code != http.StatusNoContent {
		t.Fatalf("preflight expected 204, got %d: %s", wOptions.Code, wOptions.Body.String())
	}
	allowHeaders := wOptions.Header().Get("Access-Control-Allow-Headers")
	for _, expectedH := range []string{"Content-Type", "Accept", "Idempotency-Key", "X-Order-Access-Token"} {
		if !strings.Contains(allowHeaders, expectedH) {
			t.Fatalf("Access-Control-Allow-Headers missing %s: %s", expectedH, allowHeaders)
		}
	}

	// 2. Preflight from unknown origin -> 403 ORIGIN_FORBIDDEN
	reqEvilPreflight := httptest.NewRequest(http.MethodOptions, "/api/v1/orders", nil)
	reqEvilPreflight.Header.Set("Origin", "https://evil.example")
	reqEvilPreflight.Header.Set("Access-Control-Request-Method", http.MethodPost)
	wEvilPreflight := httptest.NewRecorder()
	router.ServeHTTP(wEvilPreflight, reqEvilPreflight)

	if wEvilPreflight.Code != http.StatusForbidden {
		t.Fatalf("expected 403 on evil origin preflight, got %d", wEvilPreflight.Code)
	}

	// 3. Preflight with unknown header -> 400 INVALID_PREFLIGHT
	reqBadH := httptest.NewRequest(http.MethodOptions, "/api/v1/orders", nil)
	reqBadH.Header.Set("Origin", "https://shop.example")
	reqBadH.Header.Set("Access-Control-Request-Method", http.MethodPost)
	reqBadH.Header.Set("Access-Control-Request-Headers", "content-type, x-custom-forbidden")
	wBadH := httptest.NewRecorder()
	router.ServeHTTP(wBadH, reqBadH)

	if wBadH.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on forbidden header preflight, got %d", wBadH.Code)
	}

	// 4. Preflight on /api/v1/auth/... with idempotency-key STILL rejected -> 400
	reqAuthPreflight := httptest.NewRequest(http.MethodOptions, "/api/v1/auth/login", nil)
	reqAuthPreflight.Header.Set("Origin", "https://shop.example")
	reqAuthPreflight.Header.Set("Access-Control-Request-Method", http.MethodPost)
	reqAuthPreflight.Header.Set("Access-Control-Request-Headers", "content-type, idempotency-key")
	wAuthPreflight := httptest.NewRecorder()
	router.ServeHTTP(wAuthPreflight, reqAuthPreflight)

	if wAuthPreflight.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on auth preflight with idempotency-key, got %d", wAuthPreflight.Code)
	}

	// 5. Real request from unknown origin -> 403 ORIGIN_FORBIDDEN
	reqEvilReal := httptest.NewRequest(http.MethodPost, "/api/v1/orders/quote", strings.NewReader(`{}`))
	reqEvilReal.Header.Set("Origin", "https://evil.example")
	reqEvilReal.Header.Set("Content-Type", "application/json")
	wEvilReal := httptest.NewRecorder()
	router.ServeHTTP(wEvilReal, reqEvilReal)

	if wEvilReal.Code != http.StatusForbidden {
		t.Fatalf("expected 403 on real request with evil origin, got %d", wEvilReal.Code)
	}
	if wEvilReal.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("expected Cache-Control: no-store, got %s", wEvilReal.Header().Get("Cache-Control"))
	}
}

func TestNoSensitiveDataInLogs(t *testing.T) {
	token, fixtureHash := createGuestTokenFixture()
	quoteToken := "secret-quote-token-xyz-123456789"
	phone := "0912345678"

	orderSample := sampleOrder("690000000000000000000010", "AD-261116-TEST0001")
	service := &stubOrderService{
		createFunc: func(context.Context, apporder.Principal, string, domainorder.CheckoutInput) (domainorder.Order, bool, error) {
			return orderSample, true, nil
		},
	}

	var logBuf bytes.Buffer
	testLogger := zerolog.New(&logBuf)

	repo := &authHTTPRepository{}
	authService := appauth.NewService(repo, time.Hour)
	guard := NewAuthGuard(authService, AuthCookieConfig{Name: "session", TTL: time.Hour})

	mux := http.NewServeMux()
	WithOrders(service, guard, testLogger)(mux)
	router := NewRouter(nil, testLogger, func(m *http.ServeMux) {
		WithOrders(service, guard, testLogger)(m)
	})

	body := fmt.Sprintf(`{
		"payment_method": "cash_at_counter",
		"shipping_address": {"recipient": "Nguyen An", "phone": %q},
		"items": [{"product_id": "680000000000000000000001", "sku": "LUA-DO-M", "quantity": 1}],
		"quote_token": %q
	}`, phone, quoteToken)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/orders", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "ad577a65-cd42-46eb-ae8a-98b5f1186571")
	req.Header.Set("X-Order-Access-Token", token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	logOutput := logBuf.String()
	for _, sensitive := range []string{token, fixtureHash, quoteToken, phone} {
		if strings.Contains(logOutput, sensitive) {
			t.Fatalf("log contains sensitive value %q! Log content: %s", sensitive, logOutput)
		}
	}
}
