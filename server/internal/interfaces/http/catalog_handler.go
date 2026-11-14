package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/rs/zerolog"

	app "github.com/your-org/aodora/server/internal/application/catalog"
	model "github.com/your-org/aodora/server/internal/domain/catalog"
)

type catalogHandler struct {
	service *app.Service
	log     zerolog.Logger
}

// Log internal causes while keeping connection URI credentials and other
// sensitive values out of logs and generic HTTP error responses.
var mongodbURI = regexp.MustCompile(`(?i)mongodb(?:\+srv)?://[^\s]+`)

func safeCatalogError(err error) string {
	if err == nil {
		return ""
	}
	return mongodbURI.ReplaceAllString(err.Error(), "[mongodb-uri-redacted]")
}
func (h catalogHandler) logFailure(operation string, err error) {
	h.log.Error().Str("operation", operation).Str("cause", safeCatalogError(err)).Msg("catalog request failed")
}
func (h catalogHandler) respondFailure(w http.ResponseWriter, operation string, err error, message string) {
	h.logFailure(operation, err)
	if errors.Is(err, context.DeadlineExceeded) {
		respondError(w, http.StatusGatewayTimeout, "CATALOG_TIMEOUT", "", "Truy vấn sản phẩm quá thời gian cho phép")
		return
	}
	respondError(w, http.StatusInternalServerError, "CATALOG_UNAVAILABLE", "", message)
}

// WithCatalog keeps the public catalog API independent of auth/admin routes.
func WithCatalog(service *app.Service, logs ...zerolog.Logger) RouterOption {
	log := zerolog.Nop()
	if len(logs) > 0 {
		log = logs[0]
	}
	return func(mux *http.ServeMux) {
		handler := catalogHandler{service: service, log: log}
		mux.HandleFunc("GET /api/v1/categories", handler.categories)
		mux.HandleFunc("GET /api/v1/products", handler.products)
		mux.HandleFunc("GET /api/v1/products/{slug}", handler.product)
	}
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Field   string `json:"field,omitempty"`
}

func respondError(w http.ResponseWriter, status int, code, field, message string) {
	writeJSON(w, status, map[string]apiError{"error": {Code: code, Field: field, Message: message}})
}

func (h catalogHandler) categories(w http.ResponseWriter, r *http.Request) {
	if len(r.URL.Query()) != 0 {
		respondError(w, http.StatusBadRequest, "INVALID_QUERY", "query", "Danh sách danh mục không nhận tham số lọc")
		return
	}
	categories, err := h.service.Categories(r.Context())
	if err != nil {
		h.respondFailure(w, "categories", err, "Không thể tải danh mục")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": categories})
}

func (h catalogHandler) products(w http.ResponseWriter, r *http.Request) {
	filter, field, message := parseProductFilter(r.URL.Query())
	if field != "" {
		respondError(w, http.StatusBadRequest, "INVALID_QUERY", field, message)
		return
	}
	products, err := h.service.List(r.Context(), filter)
	if err != nil {
		h.respondFailure(w, "products", err, "Không thể tải sản phẩm")
		return
	}
	writeJSON(w, http.StatusOK, products)
}

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

func (h catalogHandler) product(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	if len(slug) > 150 || !slugPattern.MatchString(slug) {
		respondError(w, http.StatusBadRequest, "INVALID_SLUG", "slug", "Slug sản phẩm không hợp lệ")
		return
	}
	if len(r.URL.Query()) != 0 {
		respondError(w, http.StatusBadRequest, "INVALID_QUERY", "query", "Chi tiết sản phẩm không nhận tham số lọc")
		return
	}
	product, err := h.service.Detail(r.Context(), slug)
	if errors.Is(err, app.ErrProductNotFound) {
		respondError(w, http.StatusNotFound, "PRODUCT_NOT_FOUND", "slug", "Không tìm thấy sản phẩm đang bán")
		return
	}
	if err != nil {
		h.respondFailure(w, "product_detail", err, "Không thể tải sản phẩm")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": product})
}

var allowedProductParams = map[string]bool{
	"search": true, "q": true, "category": true, "min_price": true,
	"max_price": true, "color": true, "size": true, "material": true,
	"sort": true, "page": true, "page_size": true,
}

func parseProductFilter(query url.Values) (model.Filter, string, string) {
	filter := model.Filter{Sort: "newest", Page: 1, PageSize: 12}
	for key, values := range query {
		if !allowedProductParams[key] {
			return filter, key, "Tham số không được hỗ trợ"
		}
		if len(values) != 1 {
			return filter, key, "Mỗi tham số chỉ được truyền một lần"
		}
	}
	if query.Has("search") && query.Has("q") {
		return filter, "search", "Chỉ dùng một trong hai tham số search hoặc q"
	}
	filter.Search = strings.TrimSpace(query.Get("search"))
	if query.Has("q") {
		filter.Search = strings.TrimSpace(query.Get("q"))
	}
	filter.Category = strings.TrimSpace(query.Get("category"))
	filter.Color = strings.TrimSpace(query.Get("color"))
	filter.Size = strings.TrimSpace(query.Get("size"))
	filter.Material = strings.TrimSpace(query.Get("material"))
	for _, field := range []struct {
		name, value string
		max         int
	}{
		{"search", filter.Search, 120}, {"category", filter.Category, 150},
		{"color", filter.Color, 80}, {"size", filter.Size, 30},
		{"material", filter.Material, 100},
	} {
		if !utf8.ValidString(field.value) || utf8.RuneCountInString(field.value) > field.max ||
			strings.ContainsAny(field.value, "\x00\n\r") {
			return filter, field.name, "Giá trị lọc không hợp lệ hoặc quá dài"
		}
	}
	if query.Has("sort") {
		filter.Sort = query.Get("sort")
		switch filter.Sort {
		case "newest", "price_asc", "price_desc", "name_asc", "name_desc", "price-asc", "price-desc":
		default:
			return filter, "sort", "Giá trị sort không hợp lệ"
		}
	}
	if query.Has("page") {
		value, err := strconv.ParseInt(query.Get("page"), 10, 64)
		if err != nil || value < 1 || value > 200 {
			return filter, "page", "page phải là số nguyên từ 1 đến 200"
		}
		filter.Page = value
	}
	if query.Has("page_size") {
		value, err := strconv.ParseInt(query.Get("page_size"), 10, 64)
		if err != nil || value < 1 || value > 50 {
			return filter, "page_size", "page_size phải là số nguyên từ 1 đến 50"
		}
		filter.PageSize = value
	}
	if query.Has("min_price") {
		value, err := strconv.ParseInt(query.Get("min_price"), 10, 64)
		if err != nil || value < 0 {
			return filter, "min_price", "min_price phải là số nguyên không âm"
		}
		filter.MinPrice = &value
	}
	if query.Has("max_price") {
		value, err := strconv.ParseInt(query.Get("max_price"), 10, 64)
		if err != nil || value < 0 {
			return filter, "max_price", "max_price phải là số nguyên không âm"
		}
		filter.MaxPrice = &value
	}
	if (filter.Page-1)*filter.PageSize > 10000 {
		return filter, "page", "Vượt giới hạn độ sâu phân trang"
	}
	if filter.MinPrice != nil && filter.MaxPrice != nil && *filter.MinPrice > *filter.MaxPrice {
		return filter, "min_price", "min_price không được lớn hơn max_price"
	}
	return filter, "", ""
}
