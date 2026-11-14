package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	app "github.com/your-org/aodora/server/internal/application/catalog"
	model "github.com/your-org/aodora/server/internal/domain/catalog"
	"github.com/your-org/aodora/server/internal/logger"
)

type apiCatalogRepo struct{}

func (apiCatalogRepo) Categories(context.Context) ([]model.Category, error) {
	return []model.Category{{ID: "670000000000000000000001", Slug: "co-phuc", Name: "Cổ phục", IsActive: true}}, nil
}
func (apiCatalogRepo) ListProducts(context.Context, model.Filter, []string) ([]model.Product, int64, error) {
	return []model.Product{{ID: "680000000000000000000001", CategoryID: "670000000000000000000001",
		Slug: "ao-a", Name: "Áo A", Price: json.Number("2190000")}}, 1, nil
}
func (apiCatalogRepo) ProductBySlug(_ context.Context, slug string) (*model.Product, error) {
	if slug != "ao-a" {
		return nil, nil
	}
	return &model.Product{ID: "680000000000000000000001", CategoryID: "670000000000000000000001",
		Slug: "ao-a", Name: "Áo A", Price: json.Number("2190000"), Story: &model.Story{Title: "Câu chuyện của A"}}, nil
}
func (apiCatalogRepo) ReviewSummary(context.Context, string) (model.ReviewSummary, error) {
	return model.ReviewSummary{Count: 3, Average: 4.67}, nil
}

func testCatalogRouter() http.Handler {
	return NewRouter(nil, *logger.NewDiscard(), WithCatalog(app.NewService(apiCatalogRepo{})))
}

func TestPublicCatalogEndpoints(t *testing.T) {
	cases := []struct {
		path   string
		status int
		body   string
	}{
		{"/api/v1/categories", 200, "Cổ phục"},
		{"/api/v1/products?page=1&page_size=10&sort=price-asc&category=co-phuc", 200, "2190000"},
		{"/api/v1/products/ao-a", 200, "Câu chuyện của A"},
		{"/api/v1/products/missing", 404, "PRODUCT_NOT_FOUND"},
		{"/api/v1/products/BAD!", 400, "INVALID_SLUG"},
		{"/api/v1/products?page=0", 400, "INVALID_QUERY"},
		{"/api/v1/products?min_price=200&max_price=100", 400, "min_price"},
		{"/api/v1/products?page_size=51", 400, "page_size"},
		{"/api/v1/products?sort=unknown", 400, "sort"},
		{"/api/v1/products?bad=1", 400, "bad"},
		{"/api/v1/products?search=abc&q=xyz", 400, "search"},
		{"/api/v1/products?color=a&color=b", 400, "color"},
	}
	for _, tt := range cases {
		t.Run(tt.path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			testCatalogRouter().ServeHTTP(rec, httptest.NewRequest("GET", tt.path, nil))
			if rec.Code != tt.status {
				t.Fatalf("status %d wanted %d: %s", rec.Code, tt.status, rec.Body.String())
			}
			if !strings.Contains(rec.Body.String(), tt.body) {
				t.Fatalf("body %s does not contain %q", rec.Body.String(), tt.body)
			}
		})
	}
}

func TestFilterParsing(t *testing.T) {
	values := url.Values{"q": {"lụa tơ"}, "size": {"M"}, "color": {"Đỏ"},
		"min_price": {"1000000"}, "max_price": {"5000000"}, "page": {"3"}, "page_size": {"4"}}
	filter, field, message := parseProductFilter(values)
	if field != "" {
		t.Fatalf("%s %s", field, message)
	}
	if filter.Search != "lụa tơ" || filter.Page != 3 || filter.PageSize != 4 || *filter.MinPrice != 1000000 || filter.Color != "Đỏ" {
		t.Fatalf("bad filter %+v", filter)
	}
}

type failingCatalogRepo struct{ cause error }

func (r failingCatalogRepo) Categories(context.Context) ([]model.Category, error) {
	return nil, r.cause
}
func (r failingCatalogRepo) ListProducts(context.Context, model.Filter, []string) ([]model.Product, int64, error) {
	return nil, 0, r.cause
}
func (r failingCatalogRepo) ProductBySlug(context.Context, string) (*model.Product, error) {
	return nil, r.cause
}
func (r failingCatalogRepo) ReviewSummary(context.Context, string) (model.ReviewSummary, error) {
	return model.ReviewSummary{}, r.cause
}

func TestCatalogErrorsAreLoggedWithoutCredentials(t *testing.T) {
	var buffer bytes.Buffer
	log := zerolog.New(&buffer)
	// Construct a synthetic URI rather than embedding credentials in source.
	// No connection is established; this only tests safe error reporting.
	testPassword := "unit-test-credential"
	connectionURI := (&url.URL{
		Scheme: "mongodb",
		User:   url.UserPassword("fixture-user", testPassword),
		Host:   "example.invalid:27017",
		Path:   "/test-db",
	}).String()
	injected := errors.New("cannot connect " + connectionURI)
	router := NewRouter(nil, *logger.NewDiscard(), WithCatalog(app.NewService(failingCatalogRepo{cause: injected}), log))
	for _, path := range []string{"/api/v1/categories", "/api/v1/products", "/api/v1/products/test-slug"} {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("%s: %d %s", path, rec.Code, rec.Body.String())
		}
		if strings.Contains(rec.Body.String(), testPassword) || strings.Contains(rec.Body.String(), connectionURI) {
			t.Fatal("leaked connection credentials to client")
		}
	}
	logs := buffer.String()
	if !strings.Contains(logs, "catalog request failed") ||
		!strings.Contains(logs, "[mongodb-uri-redacted]") ||
		strings.Contains(logs, testPassword) || strings.Contains(logs, connectionURI) {
		t.Fatalf("missing redaction or sensitive log: %s", logs)
	}
}

func TestWriteJSONReturns500OnMarshallingFailure(t *testing.T) {
	rec := httptest.NewRecorder()
	writeJSON(rec, http.StatusOK, map[string]any{"unsupported": make(chan int)})
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("wrote %d instead of 500", rec.Code)
	}
	var result map[string]any
	if err := json.NewDecoder(io.NopCloser(strings.NewReader(rec.Body.String()))).Decode(&result); err != nil {
		t.Fatalf("invalid error response: %v", err)
	}
}

func TestDeepPaginationRejected(t *testing.T) {
	for _, path := range []string{
		"/api/v1/products?page=201", "/api/v1/products?page=200&page_size=51",
	} {
		recorder := httptest.NewRecorder()
		testCatalogRouter().ServeHTTP(recorder, httptest.NewRequest("GET", path, nil))
		if recorder.Code != 400 {
			t.Fatalf("%s: status %d", path, recorder.Code)
		}
	}
}
