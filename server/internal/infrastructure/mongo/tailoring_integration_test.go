//go:build integration

package mongo

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
	appauth "github.com/your-org/aodora/server/internal/application/auth"
	apptailoring "github.com/your-org/aodora/server/internal/application/tailoring"
	httpapi "github.com/your-org/aodora/server/internal/interfaces/http"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestTailoringIntegration_RealMongoDB(t *testing.T) {
	client, db, _ := orderTestDatabase(t)
	ctx := context.Background()

	// Clear tailoring_requests collection
	if _, err := db.Collection("tailoring_requests").DeleteMany(ctx, bson.M{}); err != nil {
		t.Fatalf("clear tailoring_requests: %v", err)
	}

	repo := NewTailoringRepository(db)
	svc := apptailoring.NewService(repo)

	// Auth setup with real AuthRepository
	authRepo := NewAuthRepository(db)
	authSvc := appauth.NewService(authRepo, 24*time.Hour)
	authGuard := httpapi.NewAuthGuard(authSvc, httpapi.AuthCookieConfig{
		Name: "aodora_session",
	})

	handler := httpapi.NewRouter(nil, zerolog.Nop(), httpapi.WithTailoring(svc, authGuard, zerolog.Nop()))

	// Create a real customer in DB for authenticated testing
	custOID := bson.NewObjectID()
	custEmail := "tailoring_test_user@example.com"
	now := time.Now().UTC()
	if _, err := db.Collection("customers").InsertOne(ctx, bson.M{
		"_id":           custOID,
		"full_name":     "Nguyễn Thị Lan",
		"email":         custEmail,
		"password_hash": "$2a$10$abcdefghijklmnopqrstuv",
		"created_at":    now,
		"updated_at":    now,
	}); err != nil {
		t.Fatalf("create test customer: %v", err)
	}

	// Create a session for this customer
	sessionToken := "tailoring-test-session-token-12345"
	digest := sha256.Sum256([]byte(sessionToken))
	tokenHash := hex.EncodeToString(digest[:])
	if err := authRepo.CreateSession(ctx, custOID.Hex(), tokenHash, now.Add(24*time.Hour)); err != nil {
		t.Fatalf("create test customer session: %v", err)
	}

	// Create a real product in DB
	prodOID := bson.NewObjectID()
	catOID := bson.NewObjectID()
	prodSlug := "ao-dai-gam-do"
	if _, err := db.Collection("products").InsertOne(ctx, bson.M{
		"_id":         prodOID,
		"category_id": catOID,
		"name":        "Áo Dài Gấm Đỏ",
		"slug":        prodSlug,
		"price":       int64(1200000),
		"stock":       int64(10),
		"is_active":   true,
		"created_at":  now,
		"updated_at":  now,
	}); err != nil {
		t.Fatalf("create test product: %v", err)
	}

	t.Run("Guest Minimal - Only Name and Phone", func(t *testing.T) {
		body := `{"full_name":"Khách Vãng Lai","phone":"0912345678"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/tailoring-requests", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d: %s", w.Code, w.Body.String())
		}

		var resp struct {
			Data struct {
				ID     string `json:"id"`
				Status string `json:"status"`
			} `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode response: %v", err)
		}

		if resp.Data.ID == "" || resp.Data.Status != "submitted" {
			t.Fatalf("unexpected response data: %+v", resp.Data)
		}

		// Verify in MongoDB
		oid, err := bson.ObjectIDFromHex(resp.Data.ID)
		if err != nil {
			t.Fatalf("invalid returned ID hex: %v", err)
		}

		var doc bson.M
		if err := db.Collection("tailoring_requests").FindOne(ctx, bson.M{"_id": oid}).Decode(&doc); err != nil {
			t.Fatalf("document not found in MongoDB: %v", err)
		}

		if doc["full_name"] != "Khách Vãng Lai" {
			t.Errorf("expected full_name 'Khách Vãng Lai', got %v", doc["full_name"])
		}
		if doc["phone"] != "0912345678" {
			t.Errorf("expected phone '0912345678', got %v", doc["phone"])
		}
		if doc["status"] != "submitted" {
			t.Errorf("expected status 'submitted', got %v", doc["status"])
		}
		if doc["customer_id"] != nil {
			t.Errorf("expected customer_id to be absent or nil, got %v", doc["customer_id"])
		}
		if doc["measurements"] != nil {
			t.Errorf("expected measurements to be absent or nil, got %v", doc["measurements"])
		}
	})

	t.Run("Missing Name or Phone - 400 and No DB Insert", func(t *testing.T) {
		countBefore, err := db.Collection("tailoring_requests").CountDocuments(ctx, bson.M{})
		if err != nil {
			t.Fatalf("count before: %v", err)
		}

		for _, tc := range []struct {
			name string
			body string
		}{
			{"empty name", `{"full_name":"","phone":"0912345678"}`},
			{"whitespace name", `{"full_name":"   ","phone":"0912345678"}`},
			{"missing name", `{"phone":"0912345678"}`},
			{"empty phone", `{"full_name":"Trần Văn B","phone":""}`},
			{"whitespace phone", `{"full_name":"Trần Văn B","phone":"   "}`},
			{"missing phone", `{"full_name":"Trần Văn B"}`},
		} {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/tailoring-requests", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("%s: expected 400, got %d: %s", tc.name, w.Code, w.Body.String())
			}
		}

		countAfter, err := db.Collection("tailoring_requests").CountDocuments(ctx, bson.M{})
		if err != nil {
			t.Fatalf("count after: %v", err)
		}
		if countAfter != countBefore {
			t.Fatalf("documents were unexpectedly inserted on validation failures! before=%d, after=%d", countBefore, countAfter)
		}
	})

	t.Run("Authenticated Customer - customer_id Linked", func(t *testing.T) {
		body := `{"full_name":"Nguyễn Thị Lan","phone":"0912345678"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/tailoring-requests", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(&http.Cookie{Name: "aodora_session", Value: sessionToken})
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d: %s", w.Code, w.Body.String())
		}

		var resp struct {
			Data struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &resp)

		oid, _ := bson.ObjectIDFromHex(resp.Data.ID)
		var doc bson.M
		if err := db.Collection("tailoring_requests").FindOne(ctx, bson.M{"_id": oid}).Decode(&doc); err != nil {
			t.Fatalf("find doc: %v", err)
		}

		if doc["customer_id"] != custOID {
			t.Errorf("expected customer_id %v, got %v", custOID, doc["customer_id"])
		}
	})

	t.Run("Invalid Cookie - 401 and No DB Insert", func(t *testing.T) {
		countBefore, _ := db.Collection("tailoring_requests").CountDocuments(ctx, bson.M{})

		body := `{"full_name":"Kẻ Giả Mạo","phone":"0912345678"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/tailoring-requests", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(&http.Cookie{Name: "aodora_session", Value: "invalid-token-xyz"})
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d: %s", w.Code, w.Body.String())
		}

		countAfter, _ := db.Collection("tailoring_requests").CountDocuments(ctx, bson.M{})
		if countAfter != countBefore {
			t.Fatalf("document was inserted despite 401!")
		}
	})

	t.Run("Valid Product Reference", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{
			"full_name":    "Khách Mua Mẫu",
			"phone":        "0912345678",
			"product_id":   prodOID.Hex(),
			"product_slug": prodSlug,
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/tailoring-requests", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
		}

		var resp struct {
			Data struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &resp)

		oid, _ := bson.ObjectIDFromHex(resp.Data.ID)
		var doc bson.M
		_ = db.Collection("tailoring_requests").FindOne(ctx, bson.M{"_id": oid}).Decode(&doc)

		if doc["product_id"] != prodOID {
			t.Errorf("expected product_id %v, got %v", prodOID, doc["product_id"])
		}
		if doc["product_slug"] != prodSlug {
			t.Errorf("expected product_slug %q, got %v", prodSlug, doc["product_slug"])
		}
	})

	t.Run("Inactive Product - 400 and No DB Insert", func(t *testing.T) {
		inactiveOID := bson.NewObjectID()
		inactiveSlug := "ao-dai-da-an"
		if _, err := db.Collection("products").InsertOne(ctx, bson.M{
			"_id":         inactiveOID,
			"category_id": catOID,
			"name":        "Áo Dài Đã Ẩn",
			"slug":        inactiveSlug,
			"price":       int64(1000000),
			"stock":       int64(1),
			"is_active":   false,
			"created_at":  now,
			"updated_at":  now,
		}); err != nil {
			t.Fatalf("create inactive product: %v", err)
		}
		before, err := db.Collection("tailoring_requests").CountDocuments(ctx, bson.M{})
		if err != nil {
			t.Fatalf("count: %v", err)
		}
		for field, body := range map[string]string{
			"product_id":   `{"full_name":"A","phone":"0912345678","product_id":"` + inactiveOID.Hex() + `"}`,
			"product_slug": `{"full_name":"A","phone":"0912345678","product_slug":"` + inactiveSlug + `"}`,
		} {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/tailoring-requests", bytes.NewBufferString(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "INVALID_FIELD") || !strings.Contains(w.Body.String(), field) {
				t.Fatalf("%s: expected 400 INVALID_FIELD, got %d: %s", field, w.Code, w.Body.String())
			}
		}
		after, err := db.Collection("tailoring_requests").CountDocuments(ctx, bson.M{})
		if err != nil {
			t.Fatalf("count: %v", err)
		}
		if after != before {
			t.Fatalf("inactive product must not insert: before=%d after=%d", before, after)
		}
	})

	t.Run("Product Conflict and Non-Existent - 400", func(t *testing.T) {
		// Non-existent product_id
		fakeID := bson.NewObjectID().Hex()
		body1, _ := json.Marshal(map[string]any{
			"full_name":  "Khách A",
			"phone":      "0912345678",
			"product_id": fakeID,
		})
		req1 := httptest.NewRequest(http.MethodPost, "/api/v1/tailoring-requests", bytes.NewReader(body1))
		req1.Header.Set("Content-Type", "application/json")
		w1 := httptest.NewRecorder()
		handler.ServeHTTP(w1, req1)
		if w1.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for non-existent product, got %d", w1.Code)
		}

		// Conflicting product_id and product_slug
		body2, _ := json.Marshal(map[string]any{
			"full_name":    "Khách A",
			"phone":        "0912345678",
			"product_id":   prodOID.Hex(),
			"product_slug": "conflicting-slug",
		})
		req2 := httptest.NewRequest(http.MethodPost, "/api/v1/tailoring-requests", bytes.NewReader(body2))
		req2.Header.Set("Content-Type", "application/json")
		w2 := httptest.NewRecorder()
		handler.ServeHTTP(w2, req2)
		if w2.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for conflicting slug, got %d", w2.Code)
		}
	})

	t.Run("Measurements - Exact Decimals, Missing Not Zero", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{
			"full_name": "Khách May Đo",
			"phone":     "0912345678",
			"notes":     "Thích tà dài thướt tha",
			"measurements": map[string]any{
				"body":              "1m62 / 52kg",
				"neck_cm":           34.5,
				"bust_cm":           86,
				"garment_length_cm": 138.5,
			},
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/tailoring-requests", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
		}

		var resp struct {
			Data struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &resp)

		oid, _ := bson.ObjectIDFromHex(resp.Data.ID)
		var doc storedTailoringRequest
		if err := db.Collection("tailoring_requests").FindOne(ctx, bson.M{"_id": oid}).Decode(&doc); err != nil {
			t.Fatalf("find tailoring request: %v", err)
		}

		if doc.Measurements == nil {
			t.Fatalf("measurements doc missing")
		}

		if doc.Measurements.Body == nil || *doc.Measurements.Body != "1m62 / 52kg" {
			t.Errorf("expected body '1m62 / 52kg', got %v", doc.Measurements.Body)
		}
		if doc.Measurements.NeckCM == nil || *doc.Measurements.NeckCM != 34.5 {
			t.Errorf("expected neck_cm 34.5, got %v", doc.Measurements.NeckCM)
		}
		if doc.Measurements.BustCM == nil || *doc.Measurements.BustCM != 86.0 {
			t.Errorf("expected bust_cm 86, got %v", doc.Measurements.BustCM)
		}
		if doc.Measurements.GarmentLengthCM == nil || *doc.Measurements.GarmentLengthCM != 138.5 {
			t.Errorf("expected garment_length_cm 138.5, got %v", doc.Measurements.GarmentLengthCM)
		}

		// Missing measurements must NOT be converted to 0
		if doc.Measurements.WaistCM != nil {
			t.Errorf("waist_cm should not exist or be nil, got %v", *doc.Measurements.WaistCM)
		}
		if doc.Measurements.HipsCM != nil {
			t.Errorf("hips_cm should not exist or be nil, got %v", *doc.Measurements.HipsCM)
		}
		if doc.Measurements.ShoulderCM != nil {
			t.Errorf("shoulder_cm should not exist or be nil, got %v", *doc.Measurements.ShoulderCM)
		}
		if doc.Measurements.SleeveCM != nil {
			t.Errorf("sleeve_cm should not exist or be nil, got %v", *doc.Measurements.SleeveCM)
		}
	})

	t.Run("Measurements - Reject Negative, Zero, and String", func(t *testing.T) {
		for _, tc := range []struct {
			name string
			meas map[string]any
		}{
			{"negative", map[string]any{"neck_cm": -10}},
			{"zero", map[string]any{"bust_cm": 0}},
			{"string", map[string]any{"waist_cm": "chữ"}},
			{"too large", map[string]any{"garment_length_cm": 9999}},
		} {
			body, _ := json.Marshal(map[string]any{
				"full_name":    "Khách Sai",
				"phone":        "0912345678",
				"measurements": tc.meas,
			})
			req := httptest.NewRequest(http.MethodPost, "/api/v1/tailoring-requests", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("%s: expected 400, got %d: %s", tc.name, w.Code, w.Body.String())
			}
		}
	})

	_ = client
}
