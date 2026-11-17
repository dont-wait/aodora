package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog"
	app "github.com/your-org/aodora/server/internal/application/order"
	domain "github.com/your-org/aodora/server/internal/domain/order"
)

// AdminOrderService exposes only the operations needed by admin order routes.
type AdminOrderService interface {
	List(context.Context, app.AdminFilter) ([]domain.Order, int64, int64, int64, error)
	Get(context.Context, string) (*domain.Order, error)
	SetStatus(context.Context, string, string) (*domain.Order, error)
	Cancel(context.Context, string) (*domain.Order, error)
	ConfirmRestock(context.Context, string) (*domain.Order, error)
	SetPayment(context.Context, string, string) (*domain.Order, error)
}

type adminOrderHandler struct {
	service AdminOrderService
	log     zerolog.Logger
}

func WithAdminOrders(service AdminOrderService, authorize AdminAuthorizer, log zerolog.Logger, allowedOrigin string) RouterOption {
	return func(mux *http.ServeMux) {
		h := adminOrderHandler{service: service, log: log}
		// Reuse the catalog guard, including origin normalization. Its existing
		// OPTIONS /api/v1/admin/{path...} registration handles preflight for both.
		guard := adminCatalogHandler{authorize: authorize, log: log, allowedOrigin: strings.TrimSuffix(strings.TrimSpace(allowedOrigin), "/")}
		for _, route := range []struct {
			pattern string
			handle  http.HandlerFunc
		}{
			{"GET /api/v1/admin/orders", h.list},
			{"GET /api/v1/admin/orders/{id}", h.get},
			{"PATCH /api/v1/admin/orders/{id}/status", h.status},
			{"POST /api/v1/admin/orders/{id}/cancel", h.cancel},
			{"POST /api/v1/admin/orders/{id}/restock", h.restock},
			{"PATCH /api/v1/admin/orders/{id}/payment", h.payment},
		} {
			mux.HandleFunc(route.pattern, guard.guard(route.handle))
		}
	}
}

type adminOrderItemDTO struct {
	orderItemResponseDTO
	FulfillmentMode string `json:"fulfillment_mode"`
}

type adminOrderDTO struct {
	orderDTO
	Items                []adminOrderItemDTO `json:"items"`
	UpdatedAt            time.Time           `json:"updated_at"`
	CustomerID           string              `json:"customer_id,omitempty"`
	IsGuest              bool                `json:"is_guest"`
	CancelledAt          *time.Time          `json:"cancelled_at,omitempty"`
	InventoryRestock     string              `json:"inventory_restock,omitempty"`
	InventoryRestockedAt *time.Time          `json:"inventory_restocked_at,omitempty"`
	PaidAt               *time.Time          `json:"paid_at,omitempty"`
	RefundedAt           *time.Time          `json:"refunded_at,omitempty"`
	AllowedActions       []string            `json:"allowed_actions"`
}

func toAdminOrderDTO(o domain.Order) adminOrderDTO {
	base := toOrderDTO(o)
	items := make([]adminOrderItemDTO, len(o.Items))
	for i, it := range o.Items {
		items[i] = adminOrderItemDTO{orderItemResponseDTO: base.Items[i], FulfillmentMode: it.FulfillmentMode}
	}
	actions := make([]string, 0, 5)
	for _, next := range []struct{ target, action string }{{domain.StatusConfirmed, "confirm"}, {domain.StatusShipping, "ship"}, {domain.StatusDelivered, "deliver"}} {
		if o.Status != next.target && domain.NextStatusAllowed(o.PaymentMethod, o.Status, next.target) {
			actions = append(actions, next.action)
		}
	}
	switch o.Status {
	case domain.StatusPending, domain.StatusConfirmed, domain.StatusShipping:
		actions = append(actions, "cancel")
	case domain.StatusCancelled:
		if o.InventoryRestock == domain.InventoryAwaitingReturn {
			actions = append(actions, "restock")
		}
	}
	if domain.NextPaymentAllowed(o.PaymentStatus, domain.PaymentPaid) && o.PaymentStatus != domain.PaymentPaid {
		actions = append(actions, "mark_paid")
	}
	if domain.NextPaymentAllowed(o.PaymentStatus, domain.PaymentRefunded) && o.PaymentStatus != domain.PaymentRefunded {
		actions = append(actions, "mark_refunded")
	}
	return adminOrderDTO{orderDTO: base, Items: items, UpdatedAt: o.UpdatedAt, CustomerID: o.CustomerID, IsGuest: o.CustomerID == "", CancelledAt: o.CancelledAt, InventoryRestock: o.InventoryRestock, InventoryRestockedAt: o.InventoryRestockedAt, PaidAt: o.PaidAt, RefundedAt: o.RefundedAt, AllowedActions: actions}
}

func decodeAdminOrderBody(w http.ResponseWriter, r *http.Request, dst any, optional bool) bool {
	raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 16<<10))
	if err != nil {
		respondError(w, 400, "INVALID_BODY", "body", "Không đọc được JSON hoặc vượt giới hạn 16 KiB")
		return false
	}
	if len(raw) == 0 && optional {
		return true
	}
	media, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || media != "application/json" {
		respondError(w, 400, "INVALID_BODY", "Content-Type", "Yêu cầu application/json")
		return false
	}
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || raw[0] != '{' {
		respondError(w, 400, "INVALID_BODY", "body", "Phải cung cấp một JSON object")
		return false
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		respondError(w, 400, "INVALID_BODY", "body", "JSON không hợp lệ hoặc có trường không được phép")
		return false
	}
	if err := dec.Decode(new(any)); !errors.Is(err, io.EOF) {
		respondError(w, 400, "INVALID_BODY", "body", "Dữ liệu JSON thừa")
		return false
	}
	return true
}

func (h adminOrderHandler) list(w http.ResponseWriter, r *http.Request) {
	query, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		respondError(w, 400, "INVALID_FIELD", "query", "Tham số truy vấn không hợp lệ")
		return
	}
	f := app.AdminFilter{Page: 1, PageSize: 20}
	for key, values := range query {
		if len(values) != 1 {
			respondError(w, 400, "INVALID_FIELD", "query", "Không được lặp tham số truy vấn")
			return
		}
		switch key {
		case "status":
			f.Status = values[0]
		case "payment_status":
			f.PaymentStatus = values[0]
		case "page", "page_size":
			n, err := strconv.ParseInt(values[0], 10, 64)
			if err != nil || n < 1 || (key == "page" && n > 100000) {
				respondError(w, 400, "INVALID_FIELD", key, "Phải là số nguyên dương; page tối đa 100000")
				return
			}
			if key == "page" {
				f.Page = n
			} else {
				f.PageSize = n
			}
		default:
			respondError(w, 400, "INVALID_FIELD", "query", "Tham số truy vấn không được hỗ trợ")
			return
		}
	}
	orders, total, page, size, err := h.service.List(r.Context(), f)
	if err != nil {
		h.respondError(w, err)
		return
	}
	data := make([]adminOrderDTO, len(orders))
	for i, o := range orders {
		data[i] = toAdminOrderDTO(o)
	}
	writeJSON(w, 200, map[string]any{"data": data, "total": total, "page": page, "page_size": size})
}

func (h adminOrderHandler) get(w http.ResponseWriter, r *http.Request) {
	o, err := h.service.Get(r.Context(), r.PathValue("id"))
	h.respond(w, o, err)
}

func (h adminOrderHandler) status(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Status *string `json:"status"`
	}
	if !decodeAdminOrderBody(w, r, &body, false) {
		return
	}
	if body.Status == nil || (*body.Status != domain.StatusConfirmed && *body.Status != domain.StatusShipping && *body.Status != domain.StatusDelivered) {
		h.respondError(w, domain.NewInvalidFieldError("status", "Trạng thái đích phải là confirmed, shipping hoặc delivered"))
		return
	}
	o, err := h.service.SetStatus(r.Context(), r.PathValue("id"), *body.Status)
	h.respond(w, o, err)
}

func (h adminOrderHandler) cancel(w http.ResponseWriter, r *http.Request) {
	if !decodeAdminOrderBody(w, r, &struct{}{}, true) {
		return
	}
	o, err := h.service.Cancel(r.Context(), r.PathValue("id"))
	h.respond(w, o, err)
}

func (h adminOrderHandler) restock(w http.ResponseWriter, r *http.Request) {
	if !decodeAdminOrderBody(w, r, &struct{}{}, true) {
		return
	}
	o, err := h.service.ConfirmRestock(r.Context(), r.PathValue("id"))
	h.respond(w, o, err)
}

func (h adminOrderHandler) payment(w http.ResponseWriter, r *http.Request) {
	var body struct {
		PaymentStatus *string `json:"payment_status"`
	}
	if !decodeAdminOrderBody(w, r, &body, false) {
		return
	}
	if body.PaymentStatus == nil || (*body.PaymentStatus != domain.PaymentPaid && *body.PaymentStatus != domain.PaymentRefunded) {
		h.respondError(w, domain.NewInvalidFieldError("payment_status", "Trạng thái thanh toán phải là paid hoặc refunded"))
		return
	}
	o, err := h.service.SetPayment(r.Context(), r.PathValue("id"), *body.PaymentStatus)
	h.respond(w, o, err)
}

func (h adminOrderHandler) respond(w http.ResponseWriter, o *domain.Order, err error) {
	if err != nil {
		h.respondError(w, err)
		return
	}
	if o == nil {
		h.respondError(w, app.ErrOrderUnavailable)
		return
	}
	writeJSON(w, 200, map[string]any{"data": toAdminOrderDTO(*o)})
}

func (h adminOrderHandler) respondError(w http.ResponseWriter, err error) {
	var invalid *domain.InvalidFieldError
	switch {
	case errors.As(err, &invalid):
		respondError(w, 400, "INVALID_FIELD", invalid.Field, invalid.Message)
	case errors.Is(err, app.ErrNotFound):
		respondError(w, 404, "NOT_FOUND", "", "Không tìm thấy đơn hàng")
	case errors.Is(err, app.ErrStateConflict):
		respondError(w, 409, "ORDER_STATE_CONFLICT", "", "Trạng thái đơn không cho phép thao tác này")
	case errors.Is(err, app.ErrPaymentConflict):
		respondError(w, 409, "PAYMENT_STATE_CONFLICT", "", "Trạng thái thanh toán không cho phép thao tác này")
	case errors.Is(err, app.ErrRestockUnavailable):
		respondError(w, 409, "RESTOCK_UNAVAILABLE", "", "Không thể hoàn tồn vì sản phẩm hoặc SKU không còn tồn tại")
	default:
		// Driver errors may contain document values. Record the error type only,
		// never its raw text, request body, headers, token, hash or phone number.
		h.log.Error().Str("error_type", fmt.Sprintf("%T", err)).Msg("admin order request failed")
		respondError(w, 503, "ORDER_UNAVAILABLE", "", "Dịch vụ đơn hàng tạm thời không khả dụng")
	}
}
