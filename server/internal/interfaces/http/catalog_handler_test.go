package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

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
