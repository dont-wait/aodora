package httpapi

import (
	"errors"
	app "github.com/your-org/aodora/server/internal/application/catalogadmin"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rs/zerolog"
)

func TestAdminCatalogFailsClosedWithoutKR01(t *testing.T) {
	var h adminCatalogHandler
	endpoints := []struct{ method, path string }{
		{"GET", "/api/v1/admin/categories"}, {"POST", "/api/v1/admin/categories"},
		{"GET", "/api/v1/admin/products"}, {"PUT", "/api/v1/admin/products/id"},
		{"POST", "/api/v1/admin/products/id/inventory/sku"},
	}
	for _, tt := range endpoints {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(tt.method, tt.path, strings.NewReader("{}"))
		h.guard(func(http.ResponseWriter, *http.Request) { t.Fatal("unverified request got through") })(w, r)
		if w.Code != http.StatusForbidden {
			t.Fatalf("%s %s: status %d", tt.method, tt.path, w.Code)
		}
	}
}
func TestAdminCatalogRejectsUnknownAndSensitiveFields(t *testing.T) {
	for _, raw := range []string{
		`{"name":"Áo","slug":"ao","category_id":"670000000000000000000001","price":10,"stock":999}`,
		`{"name":"Áo","slug":"ao","category_id":"670000000000000000000001","price":10,"rating":5}`,
		`{"name":"Áo","slug":"ao","category_id":"670000000000000000000001","price":10,"variants":[{"sku":"X","stock":4}]}`,
	} {
		r := httptest.NewRequest("PUT", "/api/v1/admin/products/id", strings.NewReader(raw))
		r.Header.Set("Content-Type", "application/json")
		var out struct {
			Name       string `json:"name"`
			Slug       string `json:"slug"`
			CategoryID string `json:"category_id"`
			Price      int64  `json:"price"`
		}
		if err := adminRequestBody(r, &out); err == nil {
			t.Fatalf("unknown field allowed: %s", raw)
		}
	}
}
func TestAdminCatalogAuthorizerUsesServerDecision(t *testing.T) {
	h := adminCatalogHandler{authorize: func(r *http.Request) bool { return r.Context() != nil }, log: zerolog.Nop()}
	called := false
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	h.guard(func(http.ResponseWriter, *http.Request) { called = true })(w, r)
	if !called {
		t.Fatal("verified admin request was blocked")
	}
}

func TestFullReplacementRejectsOmittedMetadata(t *testing.T) {
	r := httptest.NewRequest("PUT", "/api/v1/admin/products/id", strings.NewReader(`{"name":"Áo","is_active":true}`))
	r.Header.Set("Content-Type", "application/json")
	var input app.ProductInput
	err := adminRequestBody(r, &input, "name", "images", "is_active", "expected_version")
	var invalid app.Invalid
	if !errors.As(err, &invalid) || invalid.Field != "images" {
		t.Fatalf("omitted array was not rejected: %v", err)
	}
}
func TestCredentialedCORSPreflightRestrictsOrigin(t *testing.T) {
	h := adminCatalogHandler{allowedOrigin: "https://shop.example.test"}
	request := httptest.NewRequest("OPTIONS", "/api/v1/admin/products", nil)
	request.Header.Set("Origin", "https://shop.example.test")
	request.Header.Set("Access-Control-Request-Method", "PUT")
	request.Header.Set("Access-Control-Request-Headers", "content-type")
	w := httptest.NewRecorder()
	h.preflight(w, request)
	if w.Code != http.StatusNoContent || w.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Fatalf("allowed preflight: %d %v", w.Code, w.Header())
	}
	request.Header.Set("Origin", "https://attacker.example.test")
	w = httptest.NewRecorder()
	h.preflight(w, request)
	if w.Code != http.StatusForbidden || w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("untrusted origin allowed: %d %v", w.Code, w.Header())
	}
}
