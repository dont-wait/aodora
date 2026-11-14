package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/rs/zerolog"
	app "github.com/your-org/aodora/server/internal/application/catalogadmin"
)

// AdminAuthorizer is supplied by KR-01 after it verifies the user's session
// and admin role on the server. Never implement it by trusting a client header.
// Nil denies every request until that integration is available.
type AdminAuthorizer func(*http.Request) bool

type adminCatalogHandler struct {
	service       *app.Service
	authorize     AdminAuthorizer
	log           zerolog.Logger
	allowedOrigin string
}

func WithAdminCatalog(service *app.Service, authorize AdminAuthorizer, log zerolog.Logger, allowedOrigin ...string) RouterOption {
	return func(mux *http.ServeMux) {
		h := adminCatalogHandler{service: service, authorize: authorize, log: log}
		if len(allowedOrigin) > 0 {
			h.allowedOrigin = strings.TrimSuffix(strings.TrimSpace(allowedOrigin[0]), "/")
		}
		mux.HandleFunc("OPTIONS /api/v1/admin/{path...}", h.preflight)
		for _, route := range []struct {
			pattern string
			handle  http.HandlerFunc
		}{
			{"GET /api/v1/admin/categories", h.categories},
			{"POST /api/v1/admin/categories", h.createCategory},
			{"PUT /api/v1/admin/categories/{id}", h.updateCategory},
			{"GET /api/v1/admin/products", h.products},
			{"POST /api/v1/admin/products", h.createProduct},
			{"PUT /api/v1/admin/products/{id}", h.updateProduct},
			{"POST /api/v1/admin/products/{id}/variants", h.addVariant},
			{"PATCH /api/v1/admin/products/{id}/variants/{sku}", h.updateVariant},
			{"POST /api/v1/admin/products/{id}/inventory/{sku}", h.changeInventory},
		} {
			mux.HandleFunc(route.pattern, h.guard(route.handle))
		}
	}
}
func (h adminCatalogHandler) guard(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		if !h.cors(w, r) {
			respondError(w, http.StatusForbidden, "ORIGIN_FORBIDDEN", "", "Nguồn truy cập không được cấp phép")
			return
		}
		if h.authorize == nil || !h.authorize(r) {
			respondError(w, http.StatusForbidden, "ADMIN_REQUIRED", "", "Chỉ chủ cửa hàng được phép quản trị")
			return
		}
		next(w, r)
	}
}

// No wildcards or reflected arbitrary origins for credentialed requests.
// Same-origin navigation needs no CORS setup. Deployments with separate
// frontend/API hosts must set ADMIN_ALLOWED_ORIGIN to the exact frontend origin.
func (h adminCatalogHandler) cors(w http.ResponseWriter, r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	parsed, err := url.Parse(origin)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.Path != "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return false
	}
	if h.allowedOrigin == origin {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Add("Vary", "Origin")
		return true
	}
	// A browser can send Origin for a same-origin POST. A cross-origin
	// deployment must explicitly configure the public client origin.
	return parsed.Host == r.Host && (parsed.Scheme == "http" || r.TLS != nil)
}
func (h adminCatalogHandler) preflight(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Add("Vary", "Access-Control-Request-Method")
	w.Header().Add("Vary", "Access-Control-Request-Headers")
	if !h.cors(w, r) || r.Header.Get("Origin") == "" || h.allowedOrigin != r.Header.Get("Origin") {
		respondError(w, http.StatusForbidden, "ORIGIN_FORBIDDEN", "", "Nguồn truy cập không được cấp phép")
		return
	}
	method := r.Header.Get("Access-Control-Request-Method")
	if !slices.Contains([]string{"GET", "POST", "PUT", "PATCH"}, method) {
		respondError(w, http.StatusBadRequest, "INVALID_PREFLIGHT", "", "Phương thức không được hỗ trợ")
		return
	}
	for _, header := range strings.Split(r.Header.Get("Access-Control-Request-Headers"), ",") {
		header = strings.TrimSpace(strings.ToLower(header))
		if header != "" && header != "content-type" && header != "accept" {
			respondError(w, http.StatusBadRequest, "INVALID_PREFLIGHT", "", "Header không được hỗ trợ")
			return
		}
	}
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept")
	w.Header().Set("Access-Control-Max-Age", "600")
	w.WriteHeader(http.StatusNoContent)
}

func adminRequestBody(r *http.Request, out any, required ...string) error {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		return app.Invalid{Field: "Content-Type", Message: "Yêu cầu application/json"}
	}
	raw, readErr := io.ReadAll(io.LimitReader(r.Body, (1<<20)+1))
	if readErr != nil || len(raw) > 1<<20 {
		return app.Invalid{Field: "body", Message: "JSON quá lớn hoặc không đọc được"}
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		return app.Invalid{Field: "body", Message: "JSON không hợp lệ hoặc có trường không được phép: " + err.Error()}
	}
	var rest any
	if err := decoder.Decode(&rest); err != io.EOF {
		return app.Invalid{Field: "body", Message: "Chỉ chấp nhận một JSON object"}
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil || fields == nil {
		return app.Invalid{Field: "body", Message: "Phải cung cấp JSON object"}
	}
	for _, key := range required {
		value, ok := fields[key]
		if !ok || (string(value) == "null" && key != "image" && key != "compare_at_price" && key != "gallery" && key != "colors" && key != "accessories" && key != "images") {
			return app.Invalid{Field: key, Message: "Trường bắt buộc cho yêu cầu thay thế dữ liệu"}
		}
	}
	return nil
}
func (h adminCatalogHandler) respond(w http.ResponseWriter, operation string, data map[string]any, err error, status int) {
	if err != nil {
		var invalid app.Invalid
		switch {
		case errors.As(err, &invalid):
			respondError(w, http.StatusBadRequest, "INVALID_FIELD", invalid.Field, invalid.Message)
		case errors.Is(err, app.ErrNotFound):
			respondError(w, http.StatusNotFound, "NOT_FOUND", "", "Không tìm thấy dữ liệu")
		case errors.Is(err, app.ErrConflict):
			respondError(w, http.StatusConflict, "CONFLICT", "", "Dữ liệu đã thay đổi hoặc bị trùng; hãy tải lại")
		case errors.Is(err, app.ErrTransactionsRequired):
			respondError(w, http.StatusServiceUnavailable, "MONGO_TRANSACTION_REQUIRED", "", "Cần MongoDB replica set để đổi danh mục cha")
		case errors.Is(err, context.DeadlineExceeded):
			respondError(w, http.StatusGatewayTimeout, "ADMIN_TIMEOUT", "", "Truy vấn quá thời gian")
		default:
			h.log.Error().Str("operation", operation).Str("cause", safeCatalogError(err)).Msg("admin catalog operation failed")
			respondError(w, http.StatusInternalServerError, "ADMIN_CATALOG_UNAVAILABLE", "", "Không thể lưu dữ liệu")
		}
		return
	}
	writeJSON(w, status, map[string]any{"data": data})
}
func (h adminCatalogHandler) badBody(w http.ResponseWriter, r *http.Request, out any, required ...string) bool {
	if err := adminRequestBody(r, out, required...); err != nil {
		h.respond(w, "validation", nil, err, 0)
		return true
	}
	return false
}
func (h adminCatalogHandler) categories(w http.ResponseWriter, r *http.Request) {
	data, err := h.service.Categories(r.Context())
	if err != nil {
		h.respond(w, "list_categories", nil, err, 0)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": data})
}
func (h adminCatalogHandler) products(w http.ResponseWriter, r *http.Request) {
	page, pageSize := int64(1), int64(50)
	for key, values := range r.URL.Query() {
		if (key != "page" && key != "page_size") || len(values) != 1 {
			respondError(w, http.StatusBadRequest, "INVALID_QUERY", key, "Tham số không hợp lệ")
			return
		}
		n, err := strconv.ParseInt(values[0], 10, 64)
		if err != nil {
			respondError(w, http.StatusBadRequest, "INVALID_QUERY", key, "Phải là số nguyên")
			return
		}
		if key == "page" {
			page = n
		} else {
			pageSize = n
		}
	}
	data, err := h.service.Products(r.Context(), page, pageSize)
	if err != nil {
		h.respond(w, "list_products", nil, err, 0)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": data})
}
func (h adminCatalogHandler) createCategory(w http.ResponseWriter, r *http.Request) {
	var input app.CategoryInput
	if h.badBody(w, r, &input, "name", "slug", "description", "parent_id", "is_active", "sort_order", "image") {
		return
	}
	data, err := h.service.CreateCategory(r.Context(), input)
	h.respond(w, "create_category", data, err, http.StatusCreated)
}
func (h adminCatalogHandler) updateCategory(w http.ResponseWriter, r *http.Request) {
	var input app.CategoryInput
	if h.badBody(w, r, &input, "name", "slug", "description", "parent_id", "is_active", "sort_order", "image", "expected_version") {
		return
	}
	data, err := h.service.UpdateCategory(r.Context(), r.PathValue("id"), input)
	h.respond(w, "update_category", data, err, http.StatusOK)
}
func (h adminCatalogHandler) createProduct(w http.ResponseWriter, r *http.Request) {
	var input app.ProductCreate
	if h.badBody(w, r, &input, "name", "slug", "category_id", "description", "material", "price", "compare_at_price", "images", "image", "gallery", "colors", "accessories", "is_active", "variants") {
		return
	}
	data, err := h.service.CreateProduct(r.Context(), input)
	h.respond(w, "create_product", data, err, http.StatusCreated)
}
func (h adminCatalogHandler) updateProduct(w http.ResponseWriter, r *http.Request) {
	var input app.ProductInput
	if h.badBody(w, r, &input, "name", "slug", "category_id", "description", "material", "price", "compare_at_price", "images", "image", "gallery", "colors", "accessories", "is_active", "expected_version") {
		return
	}
	data, err := h.service.UpdateProduct(r.Context(), r.PathValue("id"), input)
	h.respond(w, "update_product", data, err, http.StatusOK)
}
func (h adminCatalogHandler) addVariant(w http.ResponseWriter, r *http.Request) {
	var input app.Variant
	if h.badBody(w, r, &input) {
		return
	}
	data, err := h.service.AddVariant(r.Context(), r.PathValue("id"), input)
	h.respond(w, "add_variant", data, err, http.StatusCreated)
}
func (h adminCatalogHandler) updateVariant(w http.ResponseWriter, r *http.Request) {
	var input app.VariantDetails
	if h.badBody(w, r, &input) {
		return
	}
	data, err := h.service.UpdateVariant(r.Context(), r.PathValue("id"), r.PathValue("sku"), input)
	h.respond(w, "update_variant", data, err, http.StatusOK)
}
func (h adminCatalogHandler) changeInventory(w http.ResponseWriter, r *http.Request) {
	var input app.InventoryDelta
	if h.badBody(w, r, &input) {
		return
	}
	data, err := h.service.AdjustInventory(r.Context(), r.PathValue("id"), r.PathValue("sku"), input.Delta, input.RequestID)
	h.respond(w, "change_inventory", data, err, http.StatusOK)
}
