package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/rs/zerolog"
	app "github.com/your-org/aodora/server/internal/application/review"
	modelcatalog "github.com/your-org/aodora/server/internal/domain/catalog"
	domainorder "github.com/your-org/aodora/server/internal/domain/order"
)

type reviewCatalog interface {
	Detail(context.Context, string) (*modelcatalog.Product, error)
}

type reviewOrders interface {
	List(context.Context, string, string, int64, int64) ([]domainorder.Order, int64, int64, int64, error)
}

type reviewHandler struct {
	service appService
	guard   *AuthGuard
	catalog reviewCatalog
	orders  reviewOrders
	log     zerolog.Logger
}

type appService interface {
	List(context.Context, string, int64, int64) (app.Page, error)
	Own(context.Context, string, string) (*app.OwnReview, error)
	Submit(context.Context, string, string, app.Input) (app.OwnReview, error)
	Statuses(context.Context, string, []string) ([]app.ProductReviewStatus, error)
}

// AdminReviewService exposes only the moderation operations.
type AdminReviewService interface {
	AdminList(context.Context, app.AdminFilter) (app.AdminPage, error)
	AdminDelete(context.Context, string) error
}

func WithReviews(service appService, guard *AuthGuard, catalog reviewCatalog, orders reviewOrders, log zerolog.Logger) RouterOption {
	return func(mux *http.ServeMux) {
		h := reviewHandler{service: service, guard: guard, catalog: catalog, orders: orders, log: log}
		origin := authHandler{guard: guard}
		mux.HandleFunc("OPTIONS /api/v1/products/{slug}/reviews", origin.preflight)
		mux.HandleFunc("GET /api/v1/products/{slug}/reviews", origin.withOrigin(h.public))
		mux.HandleFunc("OPTIONS /api/v1/me/reviews/{path...}", origin.preflight)
		mux.HandleFunc("GET /api/v1/me/reviews", h.protected(h.own))
		mux.HandleFunc("GET /api/v1/me/reviews/eligible-orders", h.protected(h.eligibleOrders))
		mux.HandleFunc("GET /api/v1/me/reviews/status", h.protected(h.statuses))
		mux.HandleFunc("PUT /api/v1/me/reviews/{product_id}", h.protected(h.submit))
	}
}

// WithAdminReviews registers moderation routes behind the admin guard. The
// shared OPTIONS /api/v1/admin/{path...} handler serves their preflight.
func WithAdminReviews(service AdminReviewService, guard *AuthGuard, log zerolog.Logger, allowedOrigin string) RouterOption {
	return func(mux *http.ServeMux) {
		h := adminReviewHandler{service: service, guard: guard, log: log}
		admin := adminCatalogHandler{authorize: guard.RequireAdmin, log: log, allowedOrigin: strings.TrimSuffix(strings.TrimSpace(allowedOrigin), "/")}
		mux.HandleFunc("GET /api/v1/admin/reviews", admin.guard(h.list))
		mux.HandleFunc("DELETE /api/v1/admin/reviews/{id}", admin.guard(h.delete))
	}
}

func (h reviewHandler) protected(next http.HandlerFunc) http.HandlerFunc {
	origin := authHandler{guard: h.guard}
	return origin.withOrigin(h.guard.Require(next))
}

func (h reviewHandler) public(w http.ResponseWriter, r *http.Request) {
	product, err := h.catalog.Detail(r.Context(), r.PathValue("slug"))
	if err != nil || product == nil {
		respondError(w, http.StatusNotFound, "PRODUCT_NOT_FOUND", "slug", "Không tìm thấy sản phẩm")
		return
	}
	page, size, paginationErr := reviewPagination(r)
	if paginationErr != "" {
		respondError(w, http.StatusBadRequest, "INVALID_QUERY", paginationErr, "Tham số phân trang không hợp lệ")
		return
	}
	data, err := h.service.List(r.Context(), product.ID, page, size)
	respondReview(w, h.log, data, err)
}

func (h reviewHandler) own(w http.ResponseWriter, r *http.Request) {
	productID := r.URL.Query().Get("product_id")
	if productID == "" {
		respondError(w, http.StatusBadRequest, "INVALID_FIELD", "product_id", "Sản phẩm là bắt buộc")
		return
	}
	user, _ := h.guard.Identity(r)
	data, err := h.service.Own(r.Context(), user.ID, productID)
	if errors.Is(err, app.ErrNotFound) {
		writeJSON(w, http.StatusOK, map[string]any{"data": nil})
		return
	}
	if err != nil {
		respondReview(w, h.log, nil, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": data})
}

type eligibleOrder struct {
	ID          string `json:"id"`
	OrderNumber string `json:"order_number"`
	CreatedAt   string `json:"created_at"`
}

// eligibleOrderPageLimit bounds the scan; 50 pages × 100 orders per customer.
const eligibleOrderPageLimit = 50

func (h reviewHandler) eligibleOrders(w http.ResponseWriter, r *http.Request) {
	productID := r.URL.Query().Get("product_id")
	if productID == "" {
		respondError(w, http.StatusBadRequest, "INVALID_FIELD", "product_id", "Sản phẩm là bắt buộc")
		return
	}
	user, _ := h.guard.Identity(r)
	result := make([]eligibleOrder, 0)
	for page := int64(1); page <= eligibleOrderPageLimit; page++ {
		orders, total, _, size, err := h.orders.List(r.Context(), user.ID, domainorder.StatusDelivered, page, 100)
		if err != nil {
			respondReview(w, h.log, nil, err)
			return
		}
		result = append(result, deliveredWithProduct(orders, productID)...)
		if len(orders) == 0 || page*size >= total {
			break
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": result})
}

func deliveredWithProduct(orders []domainorder.Order, productID string) []eligibleOrder {
	result := make([]eligibleOrder, 0)
	for _, order := range orders {
		if order.Status != domainorder.StatusDelivered {
			continue
		}
		for _, item := range order.Items {
			if item.ProductID == productID {
				result = append(result, eligibleOrder{ID: order.ID, OrderNumber: order.OrderNumber, CreatedAt: order.CreatedAt.UTC().Format("2006-01-02")})
				break
			}
		}
	}
	return result
}

// statuses: GET /api/v1/me/reviews/status?product_id=a&product_id=b
func (h reviewHandler) statuses(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	for key := range query {
		if key != "product_id" {
			respondError(w, http.StatusBadRequest, "INVALID_FIELD", "query", "Tham số truy vấn không được hỗ trợ")
			return
		}
	}
	user, _ := h.guard.Identity(r)
	data, err := h.service.Statuses(r.Context(), user.ID, query["product_id"])
	respondReview(w, h.log, data, err)
}

func (h reviewHandler) submit(w http.ResponseWriter, r *http.Request) {
	var input app.Input
	if !decodeReviewJSON(w, r, &input) {
		return
	}
	user, _ := h.guard.Identity(r)
	data, err := h.service.Submit(r.Context(), user.ID, r.PathValue("product_id"), input)
	respondReview(w, h.log, data, err)
}

type adminReviewHandler struct {
	service AdminReviewService
	guard   *AuthGuard
	log     zerolog.Logger
}

func (h adminReviewHandler) list(w http.ResponseWriter, r *http.Request) {
	query, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_FIELD", "query", "Tham số truy vấn không hợp lệ")
		return
	}
	filter := app.AdminFilter{Page: 1, Size: 20}
	for key, values := range query {
		if len(values) != 1 {
			respondError(w, http.StatusBadRequest, "INVALID_FIELD", "query", "Không được lặp tham số truy vấn")
			return
		}
		value := values[0]
		switch key {
		case "product_id":
			filter.ProductID = value
		case "rating", "page", "page_size":
			n, err := strconv.ParseInt(value, 10, 64)
			limits := map[string][2]int64{"rating": {1, 5}, "page": {1, 100000}, "page_size": {1, 100}}[key]
			if err != nil || n < limits[0] || n > limits[1] {
				respondError(w, http.StatusBadRequest, "INVALID_FIELD", key, "Giá trị không hợp lệ")
				return
			}
			switch key {
			case "rating":
				filter.Rating = n
			case "page":
				filter.Page = n
			default:
				filter.Size = n
			}
		default:
			respondError(w, http.StatusBadRequest, "INVALID_FIELD", "query", "Tham số truy vấn không được hỗ trợ")
			return
		}
	}
	page, err := h.service.AdminList(r.Context(), filter)
	if err != nil {
		respondReview(w, h.log, nil, err)
		return
	}
	writeJSON(w, http.StatusOK, page)
}

func (h adminReviewHandler) delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	err := h.service.AdminDelete(r.Context(), id)
	if err != nil {
		respondReview(w, h.log, nil, err)
		return
	}
	admin, _ := h.guard.Identity(r)
	// Audit trail for a permanent moderation action; no review content logged.
	h.log.Info().Str("admin_id", admin.ID).Str("review_id", id).Msg("admin deleted review")
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"id": id, "deleted": true}})
}

func reviewPagination(r *http.Request) (int64, int64, string) {
	page, size := int64(1), int64(6)
	for key := range r.URL.Query() {
		if key != "page" && key != "page_size" {
			return 0, 0, key
		}
	}
	if raw := r.URL.Query().Get("page"); raw != "" {
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || value < 1 || value > 100000 {
			return 0, 0, "page"
		}
		page = value
	}
	if raw := r.URL.Query().Get("page_size"); raw != "" {
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || value < 1 || value > 50 {
			return 0, 0, "page_size"
		}
		size = value
	}
	return page, size, ""
}

func decodeReviewJSON(w http.ResponseWriter, r *http.Request, out any) bool {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "Content-Type", "Yêu cầu application/json")
		return false
	}
	decoder := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "body", "JSON không hợp lệ")
		return false
	}
	return true
}

func respondReview(w http.ResponseWriter, log zerolog.Logger, data any, err error) {
	if err != nil {
		var invalid app.Invalid
		switch {
		case errors.As(err, &invalid):
			respondError(w, http.StatusBadRequest, "INVALID_FIELD", invalid.Field, invalid.Message)
		case errors.Is(err, app.ErrNotEligible):
			respondError(w, http.StatusForbidden, "REVIEW_NOT_ELIGIBLE", "order_id", "Bạn chỉ có thể đánh giá sản phẩm trong đơn đã nhận")
		case errors.Is(err, app.ErrNotFound):
			respondError(w, http.StatusNotFound, "NOT_FOUND", "", "Không tìm thấy đánh giá")
		case errors.Is(err, app.ErrProduct):
			respondError(w, http.StatusNotFound, "PRODUCT_NOT_FOUND", "product_id", "Không tìm thấy sản phẩm")
		default:
			log.Error().Err(err).Msg("review request failed")
			respondError(w, http.StatusServiceUnavailable, "REVIEW_UNAVAILABLE", "", "Dịch vụ đánh giá tạm thời không khả dụng")
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": data})
}
