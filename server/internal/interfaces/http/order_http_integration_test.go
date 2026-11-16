//go:build integration

package httpapi

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	appauth "github.com/your-org/aodora/server/internal/application/auth"
	apporder "github.com/your-org/aodora/server/internal/application/order"
	domainorder "github.com/your-org/aodora/server/internal/domain/order"
	"github.com/your-org/aodora/server/internal/infrastructure/migration"
	catalogMongo "github.com/your-org/aodora/server/internal/infrastructure/mongo"
	"github.com/your-org/aodora/server/internal/logger"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func httpIntegrationTestDatabase(t *testing.T) (*mongo.Client, *mongo.Database) {
	t.Helper()
	raw := os.Getenv("MONGO_TEST_URI")
	if raw == "" {
		t.Fatal("MONGO_TEST_URI must point to the isolated replica set; integration tests must not silently skip")
	}
	parsedURI, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse MONGO_TEST_URI: %v", err)
	}

	name := "aodora_http_test_" + bson.NewObjectID().Hex()
	parsedURI.Path = "/" + name
	targetURI := parsedURI.String()

	opts := options.Client().ApplyURI(raw).SetMaxPoolSize(10)
	client, err := mongo.Connect(opts)
	if err != nil {
		t.Fatalf("connect test client: %v", err)
	}
	db := client.Database(name)

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := db.Drop(ctx); err != nil {
			t.Errorf("drop isolated test database %s: %v", name, err)
		}
		_ = client.Disconnect(ctx)
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var hello bson.M
	if err := db.RunCommand(ctx, bson.D{{Key: "hello", Value: 1}}).Decode(&hello); err != nil {
		t.Fatalf("hello command: %v", err)
	}
	if hello["setName"] == nil || hello["isWritablePrimary"] != true {
		t.Fatalf("tests require a writable replica set: %v", hello)
	}

	migrationsPath := "../../../migrations"
	if err := migration.Run(targetURI, os.DirFS(migrationsPath), "up"); err != nil {
		t.Fatalf("run migrations on test db: %v", err)
	}

	return client, db
}

func seedHTTPCategory(t *testing.T, db *mongo.Database, name, slug string) bson.ObjectID {
	t.Helper()
	id := bson.NewObjectID()
	now := time.Now().UTC().Truncate(time.Millisecond)
	doc := bson.M{
		"_id":        id,
		"name":       name,
		"slug":       fmt.Sprintf("%s-%s", slug, id.Hex()[:8]),
		"is_active":  true,
		"created_at": now,
		"updated_at": now,
	}
	if _, err := db.Collection("categories").InsertOne(context.Background(), doc); err != nil {
		t.Fatalf("seedCategory failed: %v", err)
	}
	return id
}

func seedHTTPProduct(t *testing.T, db *mongo.Database, catID bson.ObjectID, name, sku string, price int64, stock int64) bson.ObjectID {
	t.Helper()
	id := bson.NewObjectID()
	now := time.Now().UTC().Truncate(time.Millisecond)
	variants := bson.A{
		bson.M{
			"sku":        sku,
			"color":      "Trắng",
			"size":       "M",
			"stock":      stock,
			"created_at": now,
			"updated_at": now,
		},
	}
	doc := bson.M{
		"_id":         id,
		"name":        name,
		"slug":        fmt.Sprintf("ao-%s", id.Hex()[:8]),
		"category_id": catID,
		"price":       price,
		"stock":       stock,
		"variants":    variants,
		"is_active":   true,
		"version":     int64(1),
		"created_at":  now,
		"updated_at":  now,
	}
	if _, err := db.Collection("products").InsertOne(context.Background(), doc); err != nil {
		t.Fatalf("seedProduct failed: %v", err)
	}
	return id
}

func generateIntegrationGuestToken() (string, string) {
	bytes32 := make([]byte, 32)
	for i := 0; i < 32; i++ {
		bytes32[i] = byte(i + 1)
	}
	raw := base64.RawURLEncoding.EncodeToString(bytes32)
	h := sha256.Sum256(bytes32)
	return raw, hex.EncodeToString(h[:])
}

func TestOrderHTTPEndToEndIntegration(t *testing.T) {
	client, db := httpIntegrationTestDatabase(t)
	catID := seedHTTPCategory(t, db, "Áo Dài", "ao-dai")
	prodID := seedHTTPProduct(t, db, catID, "Áo Dài Gấm", "GAM-TRANG-M", 500000, 10)

	secret := bytes.Repeat([]byte("s"), 32)
	signer, err := domainorder.NewQuoteSigner(secret)
	if err != nil {
		t.Fatalf("create signer: %v", err)
	}

	testCODFee, _ := domainorder.ParseMoney("17000")
	orderRepo := catalogMongo.NewOrderRepository(client, db)
	authRepo := catalogMongo.NewAuthRepository(db)
	authService := appauth.NewService(authRepo, 24*time.Hour)
	guard := NewAuthGuard(authService, AuthCookieConfig{Name: "session", TTL: 24 * time.Hour})

	// Service with COD fee configured
	orderService := apporder.NewService(orderRepo, orderRepo, signer, 15*time.Minute, &testCODFee, nil, nil)
	// Service without COD fee configured (for SHIPPING_UNAVAILABLE test)
	noCODService := apporder.NewService(orderRepo, orderRepo, signer, 15*time.Minute, nil, nil, nil)

	router := http.NewServeMux()
	WithAuth(guard)(router)
	WithOrders(orderService, guard, *logger.NewDiscard())(router)

	noCODRouter := http.NewServeMux()
	WithAuth(guard)(noCODRouter)
	WithOrders(noCODService, guard, *logger.NewDiscard())(noCODRouter)

	guestToken1, _ := generateIntegrationGuestToken()
	guestToken2 := base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{9}, 32))
	idempotencyKey := "ad577a65-cd42-46eb-ae8a-98b5f1186571"

	// 1. guest tại quầy quote -> 200
	quoteReqBody := fmt.Sprintf(`{
		"payment_method": "cash_at_counter",
		"shipping_address": {"recipient": "Nguyễn An", "phone": "0900000001"},
		"items": [{"product_id": %q, "sku": "GAM-TRANG-M", "quantity": 1}]
	}`, prodID.Hex())

	reqQuote := httptest.NewRequest(http.MethodPost, "/api/v1/orders/quote", strings.NewReader(quoteReqBody))
	reqQuote.Header.Set("Content-Type", "application/json")
	wQuote := httptest.NewRecorder()
	router.ServeHTTP(wQuote, reqQuote)

	if wQuote.Code != http.StatusOK {
		t.Fatalf("quote failed %d: %s", wQuote.Code, wQuote.Body.String())
	}
	var quoteResp struct {
		Data struct {
			QuoteToken  string `json:"quote_token"`
			ExpiresAt   string `json:"expires_at"`
			Subtotal    int64  `json:"subtotal"`
			ShippingFee int64  `json:"shipping_fee"`
			Total       int64  `json:"total"`
		} `json:"data"`
	}
	if err := json.Unmarshal(wQuote.Body.Bytes(), &quoteResp); err != nil {
		t.Fatalf("unmarshal quote resp: %v", err)
	}
	if quoteResp.Data.QuoteToken == "" || quoteResp.Data.Subtotal != 500000 || quoteResp.Data.ShippingFee != 0 || quoteResp.Data.Total != 500000 {
		t.Fatalf("unexpected quote result: %+v", quoteResp.Data)
	}
	quoteToken := quoteResp.Data.QuoteToken

	// 2. tạo 201
	createBody := fmt.Sprintf(`{
		"payment_method": "cash_at_counter",
		"shipping_address": {"recipient": "Nguyễn An", "phone": "0900000001"},
		"items": [{"product_id": %q, "sku": "GAM-TRANG-M", "quantity": 1}],
		"quote_token": %q
	}`, prodID.Hex(), quoteToken)

	reqCreate := httptest.NewRequest(http.MethodPost, "/api/v1/orders", strings.NewReader(createBody))
	reqCreate.Header.Set("Content-Type", "application/json")
	reqCreate.Header.Set("Idempotency-Key", idempotencyKey)
	reqCreate.Header.Set("X-Order-Access-Token", guestToken1)
	wCreate := httptest.NewRecorder()
	router.ServeHTTP(wCreate, reqCreate)

	if wCreate.Code != http.StatusCreated {
		t.Fatalf("create failed %d: %s", wCreate.Code, wCreate.Body.String())
	}
	var createResp struct {
		Data struct {
			ID          string `json:"id"`
			OrderNumber string `json:"order_number"`
			Status      string `json:"status"`
		} `json:"data"`
	}
	if err := json.Unmarshal(wCreate.Body.Bytes(), &createResp); err != nil {
		t.Fatalf("unmarshal create resp: %v", err)
	}
	orderID := createResp.Data.ID
	orderNumber := createResp.Data.OrderNumber
	if orderID == "" || orderNumber == "" || createResp.Data.Status != "pending" {
		t.Fatalf("unexpected order creation data: %+v", createResp.Data)
	}

	// 3. replay 200 cùng id / order_number
	reqReplay := httptest.NewRequest(http.MethodPost, "/api/v1/orders", strings.NewReader(createBody))
	reqReplay.Header.Set("Content-Type", "application/json")
	reqReplay.Header.Set("Idempotency-Key", idempotencyKey)
	reqReplay.Header.Set("X-Order-Access-Token", guestToken1)
	wReplay := httptest.NewRecorder()
	router.ServeHTTP(wReplay, reqReplay)

	if wReplay.Code != http.StatusOK {
		t.Fatalf("replay failed %d: %s", wReplay.Code, wReplay.Body.String())
	}
	var replayResp struct {
		Data struct {
			ID          string `json:"id"`
			OrderNumber string `json:"order_number"`
		} `json:"data"`
	}
	if err := json.Unmarshal(wReplay.Body.Bytes(), &replayResp); err != nil {
		t.Fatalf("unmarshal replay resp: %v", err)
	}
	if replayResp.Data.ID != orderID || replayResp.Data.OrderNumber != orderNumber {
		t.Fatalf("replay ID/OrderNumber mismatch: got %s/%s, want %s/%s", replayResp.Data.ID, replayResp.Data.OrderNumber, orderID, orderNumber)
	}

	// 4. GET bằng token đúng 200
	reqGetOK := httptest.NewRequest(http.MethodGet, "/api/v1/orders/"+orderID, nil)
	reqGetOK.Header.Set("X-Order-Access-Token", guestToken1)
	wGetOK := httptest.NewRecorder()
	router.ServeHTTP(wGetOK, reqGetOK)

	if wGetOK.Code != http.StatusOK {
		t.Fatalf("get order failed %d: %s", wGetOK.Code, wGetOK.Body.String())
	}

	// 5. token khác 404
	reqGetForbidden := httptest.NewRequest(http.MethodGet, "/api/v1/orders/"+orderID, nil)
	reqGetForbidden.Header.Set("X-Order-Access-Token", guestToken2)
	wGetForbidden := httptest.NewRecorder()
	router.ServeHTTP(wGetForbidden, reqGetForbidden)

	if wGetForbidden.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for different guest token, got %d", wGetForbidden.Code)
	}

	// 6. attempts 200
	reqAttempt := httptest.NewRequest(http.MethodGet, "/api/v1/orders/attempts/"+idempotencyKey, nil)
	reqAttempt.Header.Set("X-Order-Access-Token", guestToken1)
	wAttempt := httptest.NewRecorder()
	router.ServeHTTP(wAttempt, reqAttempt)

	if wAttempt.Code != http.StatusOK {
		t.Fatalf("attempt query failed %d: %s", wAttempt.Code, wAttempt.Body.String())
	}
	var attemptResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(wAttempt.Body.Bytes(), &attemptResp); err != nil {
		t.Fatalf("unmarshal attempt resp: %v", err)
	}
	if attemptResp.Data.ID != orderID {
		t.Fatalf("attempt returned wrong ID: got %s, want %s", attemptResp.Data.ID, orderID)
	}

	// 7. khác payload cùng key 409 IDEMPOTENCY_CONFLICT
	diffBody := fmt.Sprintf(`{
		"payment_method": "cash_at_counter",
		"shipping_address": {"recipient": "Nguyễn An Khác", "phone": "0900000002"},
		"items": [{"product_id": %q, "sku": "GAM-TRANG-M", "quantity": 1}],
		"quote_token": %q
	}`, prodID.Hex(), quoteToken)

	reqConflict := httptest.NewRequest(http.MethodPost, "/api/v1/orders", strings.NewReader(diffBody))
	reqConflict.Header.Set("Content-Type", "application/json")
	reqConflict.Header.Set("Idempotency-Key", idempotencyKey)
	reqConflict.Header.Set("X-Order-Access-Token", guestToken1)
	wConflict := httptest.NewRecorder()
	router.ServeHTTP(wConflict, reqConflict)

	if wConflict.Code != http.StatusConflict {
		t.Fatalf("expected 409 IDEMPOTENCY_CONFLICT, got %d: %s", wConflict.Code, wConflict.Body.String())
	}
	var conflictResp map[string]apiError
	_ = json.Unmarshal(wConflict.Body.Bytes(), &conflictResp)
	if conflictResp["error"].Code != "IDEMPOTENCY_CONFLICT" {
		t.Fatalf("expected code IDEMPOTENCY_CONFLICT, got %s", conflictResp["error"].Code)
	}

	// 8. bank_transfer 400 INVALID_FIELD
	bankBody := fmt.Sprintf(`{
		"payment_method": "bank_transfer",
		"shipping_address": {"recipient": "Nguyễn An", "phone": "0900000001"},
		"items": [{"product_id": %q, "sku": "GAM-TRANG-M", "quantity": 1}],
		"quote_token": %q
	}`, prodID.Hex(), quoteToken)

	reqBank := httptest.NewRequest(http.MethodPost, "/api/v1/orders", strings.NewReader(bankBody))
	reqBank.Header.Set("Content-Type", "application/json")
	reqBank.Header.Set("Idempotency-Key", "b0000000-0000-4000-8000-000000000001")
	reqBank.Header.Set("X-Order-Access-Token", guestToken1)
	wBank := httptest.NewRecorder()
	router.ServeHTTP(wBank, reqBank)

	if wBank.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for bank_transfer, got %d: %s", wBank.Code, wBank.Body.String())
	}

	// 9. COD khi chưa cấu hình phí 503 SHIPPING_UNAVAILABLE
	codBody := fmt.Sprintf(`{
		"payment_method": "cod",
		"shipping_address": {"recipient": "Nguyễn An", "phone": "0900000001", "line1": "12 Đường A", "city": "TP.HCM"},
		"items": [{"product_id": %q, "sku": "GAM-TRANG-M", "quantity": 1}]
	}`, prodID.Hex())

	reqNoCOD := httptest.NewRequest(http.MethodPost, "/api/v1/orders/quote", strings.NewReader(codBody))
	reqNoCOD.Header.Set("Content-Type", "application/json")
	wNoCOD := httptest.NewRecorder()
	noCODRouter.ServeHTTP(wNoCOD, reqNoCOD)

	if wNoCOD.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 SHIPPING_UNAVAILABLE, got %d: %s", wNoCOD.Code, wNoCOD.Body.String())
	}
	var noCODResp map[string]apiError
	_ = json.Unmarshal(wNoCOD.Body.Bytes(), &noCODResp)
	if noCODResp["error"].Code != "SHIPPING_UNAVAILABLE" {
		t.Fatalf("expected code SHIPPING_UNAVAILABLE, got %s", noCODResp["error"].Code)
	}

	// 10. tài khoản (đăng ký/đăng nhập qua route auth thật) tạo đơn COD với phí test
	// Register customer A
	regABody := `{"full_name":"Khách Hàng A","email":"customer_a@example.com","password":"password-123456"}`
	reqRegA := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(regABody))
	reqRegA.Header.Set("Content-Type", "application/json")
	wRegA := httptest.NewRecorder()
	router.ServeHTTP(wRegA, reqRegA)
	if wRegA.Code != http.StatusCreated {
		t.Fatalf("register customer A failed %d: %s", wRegA.Code, wRegA.Body.String())
	}

	// Login customer A
	loginABody := `{"email":"customer_a@example.com","password":"password-123456"}`
	reqLoginA := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(loginABody))
	reqLoginA.Header.Set("Content-Type", "application/json")
	wLoginA := httptest.NewRecorder()
	router.ServeHTTP(wLoginA, reqLoginA)
	if wLoginA.Code != http.StatusOK {
		t.Fatalf("login customer A failed %d: %s", wLoginA.Code, wLoginA.Body.String())
	}
	cookieA := wLoginA.Result().Cookies()[0]

	// Quote COD with test fee
	reqCODQuote := httptest.NewRequest(http.MethodPost, "/api/v1/orders/quote", strings.NewReader(codBody))
	reqCODQuote.Header.Set("Content-Type", "application/json")
	reqCODQuote.AddCookie(cookieA)
	wCODQuote := httptest.NewRecorder()
	router.ServeHTTP(wCODQuote, reqCODQuote)
	if wCODQuote.Code != http.StatusOK {
		t.Fatalf("quote COD for customer A failed %d: %s", wCODQuote.Code, wCODQuote.Body.String())
	}
	var codQuoteResp struct {
		Data struct {
			QuoteToken  string `json:"quote_token"`
			Subtotal    int64  `json:"subtotal"`
			ShippingFee int64  `json:"shipping_fee"`
			Total       int64  `json:"total"`
		} `json:"data"`
	}
	if err := json.Unmarshal(wCODQuote.Body.Bytes(), &codQuoteResp); err != nil {
		t.Fatalf("unmarshal COD quote: %v", err)
	}
	if codQuoteResp.Data.ShippingFee != 17000 || codQuoteResp.Data.Total != 517000 {
		t.Fatalf("unexpected COD quote with test fee: %+v", codQuoteResp.Data)
	}

	// Create COD order for customer A
	createCODBody := fmt.Sprintf(`{
		"payment_method": "cod",
		"shipping_address": {"recipient": "Nguyễn An", "phone": "0900000001", "line1": "12 Đường A", "city": "TP.HCM"},
		"items": [{"product_id": %q, "sku": "GAM-TRANG-M", "quantity": 1}],
		"quote_token": %q
	}`, prodID.Hex(), codQuoteResp.Data.QuoteToken)

	reqCreateA := httptest.NewRequest(http.MethodPost, "/api/v1/orders", strings.NewReader(createCODBody))
	reqCreateA.Header.Set("Content-Type", "application/json")
	reqCreateA.Header.Set("Idempotency-Key", "c0000000-0000-4000-8000-000000000001")
	reqCreateA.AddCookie(cookieA)
	wCreateA := httptest.NewRecorder()
	router.ServeHTTP(wCreateA, reqCreateA)
	if wCreateA.Code != http.StatusCreated {
		t.Fatalf("create COD order A failed %d: %s", wCreateA.Code, wCreateA.Body.String())
	}
	var orderAResp struct {
		Data struct {
			ID          string `json:"id"`
			OrderNumber string `json:"order_number"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wCreateA.Body.Bytes(), &orderAResp)
	orderAID := orderAResp.Data.ID

	// 11. GET /api/v1/orders thấy đúng đơn mình
	reqListA := httptest.NewRequest(http.MethodGet, "/api/v1/orders", nil)
	reqListA.AddCookie(cookieA)
	wListA := httptest.NewRecorder()
	router.ServeHTTP(wListA, reqListA)
	if wListA.Code != http.StatusOK {
		t.Fatalf("list orders A failed %d: %s", wListA.Code, wListA.Body.String())
	}
	var listAResp struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
		Total int64 `json:"total"`
	}
	if err := json.Unmarshal(wListA.Body.Bytes(), &listAResp); err != nil {
		t.Fatalf("unmarshal list A: %v", err)
	}
	if listAResp.Total != 1 || len(listAResp.Data) != 1 || listAResp.Data[0].ID != orderAID {
		t.Fatalf("expected 1 order with ID %s in list A, got: %+v", orderAID, listAResp)
	}

	// 12. tài khoản B đọc đơn A ⇒ 404
	// Register customer B
	regBBody := `{"full_name":"Khách Hàng B","email":"customer_b@example.com","password":"password-123456"}`
	reqRegB := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(regBBody))
	reqRegB.Header.Set("Content-Type", "application/json")
	wRegB := httptest.NewRecorder()
	router.ServeHTTP(wRegB, reqRegB)
	if wRegB.Code != http.StatusCreated {
		t.Fatalf("register customer B failed %d: %s", wRegB.Code, wRegB.Body.String())
	}

	// Login customer B
	loginBBody := `{"email":"customer_b@example.com","password":"password-123456"}`
	reqLoginB := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(loginBBody))
	reqLoginB.Header.Set("Content-Type", "application/json")
	wLoginB := httptest.NewRecorder()
	router.ServeHTTP(wLoginB, reqLoginB)
	if wLoginB.Code != http.StatusOK {
		t.Fatalf("login customer B failed %d: %s", wLoginB.Code, wLoginB.Body.String())
	}
	cookieB := wLoginB.Result().Cookies()[0]

	// Customer B reads Order A -> 404
	reqBReadsA := httptest.NewRequest(http.MethodGet, "/api/v1/orders/"+orderAID, nil)
	reqBReadsA.AddCookie(cookieB)
	wBReadsA := httptest.NewRecorder()
	router.ServeHTTP(wBReadsA, reqBReadsA)

	if wBReadsA.Code != http.StatusNotFound {
		t.Fatalf("expected 404 when Customer B reads Order A, got %d: %s", wBReadsA.Code, wBReadsA.Body.String())
	}
}
