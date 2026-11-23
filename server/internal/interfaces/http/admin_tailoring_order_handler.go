package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/rs/zerolog"
	app "github.com/your-org/aodora/server/internal/application/tailoring"
	domainorder "github.com/your-org/aodora/server/internal/domain/order"
	domain "github.com/your-org/aodora/server/internal/domain/tailoring"
)

// AdminTailoringOrderService creates the custom order of a tailoring request.
type AdminTailoringOrderService interface {
	Create(ctx context.Context, tailoringRequestID string, in app.CustomOrderInput) (order *domainorder.Order, created bool, err error)
}

type adminTailoringOrderHandler struct {
	service AdminTailoringOrderService
	log     zerolog.Logger
}

// WithAdminTailoringOrders registers POST /api/v1/admin/tailoring-requests/{id}/order
// behind the shared admin guard.
func WithAdminTailoringOrders(service AdminTailoringOrderService, authorize AdminAuthorizer, log zerolog.Logger, allowedOrigin string) RouterOption {
	return func(mux *http.ServeMux) {
		h := adminTailoringOrderHandler{service: service, log: log}
		guard := adminCatalogHandler{authorize: authorize, log: log, allowedOrigin: strings.TrimSuffix(strings.TrimSpace(allowedOrigin), "/")}
		mux.HandleFunc("POST /api/v1/admin/tailoring-requests/{id}/order", guard.guard(h.create))
	}
}

type customOrderBodyDTO struct {
	ProductID       *string         `json:"product_id"`
	UnitPrice       json.RawMessage `json:"unit_price"`
	Quantity        *int64          `json:"quantity"`
	PaymentMethod   *string         `json:"payment_method"`
	ShippingAddress *struct {
		Recipient string `json:"recipient"`
		Phone     string `json:"phone"`
		Line1     string `json:"line1"`
		Ward      string `json:"ward"`
		District  string `json:"district"`
		City      string `json:"city"`
	} `json:"shipping_address"`
	Note *string `json:"note"`
}

func (h adminTailoringOrderHandler) create(w http.ResponseWriter, r *http.Request) {
	var body customOrderBodyDTO
	if !decodeAdminOrderBody(w, r, &body, false) {
		return
	}
	raw := bytes.TrimSpace(body.UnitPrice)
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		h.respondError(w, domain.NewInvalidFieldError("unit_price", "unit_price là bắt buộc"))
		return
	}
	var price string
	if raw[0] == '"' {
		if err := json.Unmarshal(raw, &price); err != nil {
			h.respondError(w, domain.NewInvalidFieldError("unit_price", "Giá phải là số thập phân dương hợp lệ"))
			return
		}
	} else {
		price = string(raw) // exact literal: never goes through float64
	}
	if body.PaymentMethod == nil {
		h.respondError(w, domain.NewInvalidFieldError("payment_method", "payment_method là bắt buộc (cod hoặc cash_at_counter)"))
		return
	}
	in := app.CustomOrderInput{ProductID: body.ProductID, UnitPrice: price, Quantity: body.Quantity, PaymentMethod: *body.PaymentMethod}
	if body.Note != nil {
		in.Note = *body.Note
	}
	if a := body.ShippingAddress; a != nil {
		in.Address = domainorder.ShippingAddressInput{Recipient: a.Recipient, Phone: a.Phone, Line1: a.Line1, Ward: a.Ward, District: a.District, City: a.City}
	}
	o, created, err := h.service.Create(r.Context(), r.PathValue("id"), in)
	if err != nil {
		h.respondError(w, err)
		return
	}
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	writeJSON(w, status, map[string]any{"data": toAdminOrderDTO(*o), "created": created})
}

func (h adminTailoringOrderHandler) respondError(w http.ResponseWriter, err error) {
	var invalid *domain.InvalidFieldError
	switch {
	case errors.As(err, &invalid):
		respondError(w, 400, "INVALID_FIELD", invalid.Field, invalid.Message)
	case errors.Is(err, app.ErrNotFound):
		respondError(w, 404, "NOT_FOUND", "", "Không tìm thấy yêu cầu may đo")
	case errors.Is(err, app.ErrStateConflict):
		respondError(w, 409, "TAILORING_STATE_CONFLICT", "", "Chỉ tạo đơn khi số đo đã chốt hoặc đang thực hiện")
	case errors.Is(err, app.ErrMeasurementIncomplete):
		respondError(w, 409, "MEASUREMENT_INCOMPLETE", "measurements", "Cần đủ bảy số đo (cm) trước khi tạo đơn")
	case errors.Is(err, app.ErrCustomOrderExists):
		respondError(w, 409, "CUSTOM_ORDER_EXISTS", "", "Yêu cầu này đã có đơn custom với nội dung khác")
	case errors.Is(err, app.ErrShippingUnavailable):
		respondError(w, 503, "SHIPPING_UNAVAILABLE", "", "Phương thức vận chuyển tạm thời không khả dụng")
	default:
		h.log.Error().Str("error_type", fmt.Sprintf("%T", err)).Msg("admin custom order request failed")
		respondError(w, 503, "TAILORING_UNAVAILABLE", "", "Dịch vụ may đo tạm thời không khả dụng")
	}
}
