package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog"
	appauth "github.com/your-org/aodora/server/internal/application/auth"
	apporder "github.com/your-org/aodora/server/internal/application/order"
	domainorder "github.com/your-org/aodora/server/internal/domain/order"
)

// OrderService defines the subset of *apporder.Service required by orderHandler.
type OrderService interface {
	Quote(ctx context.Context, input domainorder.CheckoutInput) (*apporder.QuoteResult, error)
	Create(ctx context.Context, principal apporder.Principal, requestID string, input domainorder.CheckoutInput) (domainorder.Order, bool, error)
	Get(ctx context.Context, viewer apporder.Viewer, id string) (*domainorder.Order, error)
	ConfirmReceived(ctx context.Context, viewer apporder.Viewer, id string) (*domainorder.Order, error)
	List(ctx context.Context, customerID, status string, page, pageSize int64) ([]domainorder.Order, int64, int64, int64, error)
	Attempt(ctx context.Context, viewer apporder.Viewer, requestID string) (*domainorder.Order, error)
}

type orderHandler struct {
	service OrderService
	guard   *AuthGuard
	log     zerolog.Logger
}

// WithOrders registers order checkout, quote, and query routes on the router.
func WithOrders(service OrderService, guard *AuthGuard, log zerolog.Logger) RouterOption {
	return func(mux *http.ServeMux) {
		h := orderHandler{service: service, guard: guard, log: log}
		mux.HandleFunc("OPTIONS /api/v1/orders", h.preflight)
		mux.HandleFunc("OPTIONS /api/v1/orders/{path...}", h.preflight)
		mux.HandleFunc("POST /api/v1/orders/quote", h.wrap(h.quote))
		mux.HandleFunc("POST /api/v1/orders", h.wrap(h.create))
		mux.HandleFunc("GET /api/v1/orders", h.wrap(h.list))
		mux.HandleFunc("GET /api/v1/orders/attempts/{request_id}", h.wrap(h.attempt))
		mux.HandleFunc("GET /api/v1/orders/{id}", h.wrap(h.get))
		mux.HandleFunc("POST /api/v1/orders/{id}/received", h.wrap(h.received))
	}
}

func (h orderHandler) preflight(w http.ResponseWriter, r *http.Request) {
	h.guard.handlePreflight(w, r, []string{"idempotency-key", "x-order-access-token"}, "Content-Type, Accept, Idempotency-Key, X-Order-Access-Token")
}

func (h orderHandler) wrap(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		if !h.guard.cors(w, r) {
			respondError(w, http.StatusForbidden, "ORIGIN_FORBIDDEN", "", "Nguồn truy cập không được cấp phép")
			return
		}
		next(w, r)
	}
}

type orderAuth struct {
	User      appauth.User
	IsAuth    bool
	GuestHash string
}

func (h orderHandler) resolveAuth(w http.ResponseWriter, r *http.Request) (*orderAuth, bool) {
	user, presented, ok := h.guard.Session(r)
	if presented && !ok {
		respondError(w, http.StatusUnauthorized, "AUTH_REQUIRED", "", "Vui lòng đăng nhập")
		return nil, false
	}
	var guestHash string
	if rawToken := r.Header.Get("X-Order-Access-Token"); rawToken != "" {
		hash, _, err := domainorder.ValidateGuestAccessToken(rawToken)
		if err != nil {
			respondError(w, http.StatusBadRequest, "INVALID_FIELD", "X-Order-Access-Token", "Mã truy cập khách không hợp lệ")
			return nil, false
		}
		guestHash = hash
	} else if _, present := r.Header["X-Order-Access-Token"]; present {
		respondError(w, http.StatusBadRequest, "INVALID_FIELD", "X-Order-Access-Token", "Mã truy cập khách không hợp lệ")
		return nil, false
	}
	return &orderAuth{
		User:      user,
		IsAuth:    ok,
		GuestHash: guestHash,
	}, true
}

// Request and response DTOs

type addressDTO struct {
	Recipient *string `json:"recipient"`
	Phone     *string `json:"phone"`
	Line1     *string `json:"line1"`
	Ward      *string `json:"ward"`
	District  *string `json:"district"`
	City      *string `json:"city"`
}

type itemDTO struct {
	ProductID *string `json:"product_id"`
	SKU       *string `json:"sku"`
	Quantity  any     `json:"quantity"`
}

type quoteRequestDTO struct {
	PaymentMethod   *string     `json:"payment_method"`
	ShippingAddress *addressDTO `json:"shipping_address"`
	Items           []itemDTO   `json:"items"`
	Note            *string     `json:"note"`
}

type createRequestDTO struct {
	PaymentMethod   *string     `json:"payment_method"`
	ShippingAddress *addressDTO `json:"shipping_address"`
	Items           []itemDTO   `json:"items"`
	Note            *string     `json:"note"`
	QuoteToken      *string     `json:"quote_token"`
}

type quoteResponseItemDTO struct {
	ProductID string            `json:"product_id"`
	SKU       string            `json:"sku"`
	Quantity  int64             `json:"quantity"`
	UnitPrice domainorder.Money `json:"unit_price"`
}

type quoteResponseDTO struct {
	QuoteToken  string                 `json:"quote_token"`
	ExpiresAt   string                 `json:"expires_at"`
	Subtotal    domainorder.Money      `json:"subtotal"`
	ShippingFee domainorder.Money      `json:"shipping_fee"`
	Total       domainorder.Money      `json:"total"`
	Items       []quoteResponseItemDTO `json:"items"`
}

type orderAddressDTO struct {
	Recipient string `json:"recipient"`
	Phone     string `json:"phone"`
	Line1     string `json:"line1,omitempty"`
	Ward      string `json:"ward,omitempty"`
	District  string `json:"district,omitempty"`
	City      string `json:"city,omitempty"`
}

type orderItemResponseDTO struct {
	ProductID   string            `json:"product_id"`
	ProductName string            `json:"product_name"`
	SKU         string            `json:"sku"`
	Color       string            `json:"color,omitempty"`
	Size        string            `json:"size,omitempty"`
	Quantity    int64             `json:"quantity"`
	UnitPrice   domainorder.Money `json:"unit_price"`
}

type orderDTO struct {
	ID              string                 `json:"id"`
	OrderNumber     string                 `json:"order_number"`
	Status          string                 `json:"status"`
	PaymentMethod   string                 `json:"payment_method"`
	PaymentStatus   string                 `json:"payment_status"`
	Subtotal        domainorder.Money      `json:"subtotal"`
	ShippingFee     domainorder.Money      `json:"shipping_fee"`
	Total           domainorder.Money      `json:"total"`
	ShippingAddress orderAddressDTO        `json:"shipping_address"`
	Items           []orderItemResponseDTO `json:"items"`
	Note            string                 `json:"note,omitempty"`
	CreatedAt       string                 `json:"created_at"`
	DeliveredAt     *time.Time             `json:"delivered_at,omitempty"`
}

func toOrderDTO(o domainorder.Order) orderDTO {
	items := make([]orderItemResponseDTO, len(o.Items))
	for i, it := range o.Items {
		items[i] = orderItemResponseDTO{
			ProductID:   it.ProductID,
			ProductName: it.ProductName,
			SKU:         it.SKU,
			Color:       it.Color,
			Size:        it.Size,
			Quantity:    it.Quantity,
			UnitPrice:   it.UnitPrice,
		}
	}
	return orderDTO{
		ID:            o.ID,
		OrderNumber:   o.OrderNumber,
		Status:        o.Status,
		PaymentMethod: o.PaymentMethod,
		PaymentStatus: o.PaymentStatus,
		Subtotal:      o.Subtotal,
		ShippingFee:   o.ShippingFee,
		Total:         o.Total,
		ShippingAddress: orderAddressDTO{
			Recipient: o.ShippingAddress.Recipient,
			Phone:     o.ShippingAddress.Phone,
			Line1:     o.ShippingAddress.Line1,
			Ward:      o.ShippingAddress.Ward,
			District:  o.ShippingAddress.District,
			City:      o.ShippingAddress.City,
		},
		Items:       items,
		DeliveredAt: o.DeliveredAt,
		Note:        o.Note,
		CreatedAt:   o.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func decodeOrderJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "Content-Type", "Yêu cầu application/json")
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	dec := json.NewDecoder(r.Body)
	dec.UseNumber()
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			respondError(w, http.StatusBadRequest, "INVALID_BODY", "body", "Dung lượng yêu cầu vượt quá giới hạn 64 KiB")
			return false
		}
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "body", "Dữ liệu JSON không hợp lệ")
		return false
	}
	var extra json.RawMessage
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "body", "Dữ liệu JSON thừa")
		return false
	}
	return true
}

func toCheckoutInput(paymentMethod *string, addr *addressDTO, items []itemDTO, note *string, quoteToken *string) (domainorder.CheckoutInput, error) {
	var out domainorder.CheckoutInput
	if paymentMethod != nil {
		out.PaymentMethod = *paymentMethod
	}
	if addr != nil {
		if addr.Recipient != nil {
			out.ShippingAddress.Recipient = *addr.Recipient
		}
		if addr.Phone != nil {
			out.ShippingAddress.Phone = *addr.Phone
		}
		if addr.Line1 != nil {
			out.ShippingAddress.Line1 = *addr.Line1
		}
		if addr.Ward != nil {
			out.ShippingAddress.Ward = *addr.Ward
		}
		if addr.District != nil {
			out.ShippingAddress.District = *addr.District
		}
		if addr.City != nil {
			out.ShippingAddress.City = *addr.City
		}
	}
	if note != nil {
		out.Note = *note
	}
	if quoteToken != nil {
		out.QuoteToken = *quoteToken
	}
	out.Items = make([]domainorder.OrderItemInput, len(items))
	for i, it := range items {
		if it.ProductID != nil {
			out.Items[i].ProductID = *it.ProductID
		}
		if it.SKU != nil {
			out.Items[i].SKU = *it.SKU
		}
		switch v := it.Quantity.(type) {
		case json.Number:
			s := v.String()
			if strings.ContainsAny(s, ".eE") {
				return domainorder.CheckoutInput{}, domainorder.NewInvalidFieldError(fmt.Sprintf("items[%d].quantity", i), "Số lượng sản phẩm phải là số nguyên dương")
			}
			n, err := strconv.ParseInt(s, 10, 64)
			if err != nil || n <= 0 {
				return domainorder.CheckoutInput{}, domainorder.NewInvalidFieldError(fmt.Sprintf("items[%d].quantity", i), "Số lượng sản phẩm phải là số nguyên dương")
			}
			out.Items[i].Quantity = n
		default:
			return domainorder.CheckoutInput{}, domainorder.NewInvalidFieldError(fmt.Sprintf("items[%d].quantity", i), "Số lượng sản phẩm phải là số nguyên dương")
		}
	}
	return out, nil
}

// Handler implementations

func (h orderHandler) quote(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.resolveAuth(w, r); !ok {
		return
	}
	var req quoteRequestDTO
	if !decodeOrderJSON(w, r, &req) {
		return
	}
	input, inputErr := toCheckoutInput(req.PaymentMethod, req.ShippingAddress, req.Items, req.Note, nil)
	if inputErr != nil {
		h.respondError(w, inputErr)
		return
	}
	res, err := h.service.Quote(r.Context(), input)
	if err != nil {
		h.respondError(w, err)
		return
	}
	items := make([]quoteResponseItemDTO, len(res.Items))
	for i, it := range res.Items {
		items[i] = quoteResponseItemDTO{
			ProductID: it.ProductID,
			SKU:       it.SKU,
			Quantity:  it.Quantity,
			UnitPrice: it.UnitPrice,
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"data": quoteResponseDTO{
			QuoteToken:  res.QuoteToken,
			ExpiresAt:   res.ExpiresAt.UTC().Format(time.RFC3339),
			Subtotal:    res.Subtotal,
			ShippingFee: res.ShippingFee,
			Total:       res.Total,
			Items:       items,
		},
	})
}

func (h orderHandler) create(w http.ResponseWriter, r *http.Request) {
	auth, ok := h.resolveAuth(w, r)
	if !ok {
		return
	}
	if auth.IsAuth && auth.GuestHash != "" {
		respondError(w, http.StatusBadRequest, "INVALID_FIELD", "X-Order-Access-Token", "Không được gửi mã truy cập khách khi đã đăng nhập")
		return
	}
	if !auth.IsAuth && auth.GuestHash == "" {
		respondError(w, http.StatusBadRequest, "INVALID_FIELD", "X-Order-Access-Token", "Yêu cầu đăng nhập hoặc mã truy cập khách")
		return
	}
	var principal apporder.Principal
	if auth.IsAuth {
		principal.CustomerID = auth.User.ID
	} else {
		principal.GuestHash = auth.GuestHash
	}

	idempotencyKey := r.Header.Get("Idempotency-Key")
	canonicalKey, err := domainorder.ValidateIdempotencyKey(idempotencyKey)
	if err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_FIELD", "Idempotency-Key", "Khóa chống trùng lặp không hợp lệ")
		return
	}

	var req createRequestDTO
	if !decodeOrderJSON(w, r, &req) {
		return
	}
	input, inputErr := toCheckoutInput(req.PaymentMethod, req.ShippingAddress, req.Items, req.Note, req.QuoteToken)
	if inputErr != nil {
		h.respondError(w, inputErr)
		return
	}

	order, created, serviceErr := h.service.Create(r.Context(), principal, canonicalKey, input)
	if serviceErr != nil {
		h.respondError(w, serviceErr)
		return
	}
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	writeJSON(w, status, map[string]any{"data": toOrderDTO(order)})
}

func (h orderHandler) get(w http.ResponseWriter, r *http.Request) {
	auth, ok := h.resolveAuth(w, r)
	if !ok {
		return
	}
	if !auth.IsAuth && auth.GuestHash == "" {
		respondError(w, http.StatusNotFound, "NOT_FOUND", "", "Không tìm thấy đơn hàng")
		return
	}
	viewer := apporder.Viewer{
		GuestHash: auth.GuestHash,
	}
	if auth.IsAuth {
		viewer.CustomerID = auth.User.ID
		viewer.IsAdmin = (auth.User.Role == "admin")
	}
	id := r.PathValue("id")
	order, err := h.service.Get(r.Context(), viewer, id)
	if err != nil {
		h.respondError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": toOrderDTO(*order)})
}

// received is the buyer confirming receipt (and payment) of their order.
func (h orderHandler) received(w http.ResponseWriter, r *http.Request) {
	auth, ok := h.resolveAuth(w, r)
	if !ok {
		return
	}
	if !auth.IsAuth && auth.GuestHash == "" {
		respondError(w, http.StatusNotFound, "NOT_FOUND", "", "Không tìm thấy đơn hàng")
		return
	}
	viewer := apporder.Viewer{GuestHash: auth.GuestHash}
	if auth.IsAuth {
		viewer.CustomerID = auth.User.ID
	}
	order, err := h.service.ConfirmReceived(r.Context(), viewer, r.PathValue("id"))
	if err != nil {
		h.respondError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": toOrderDTO(*order)})
}

func (h orderHandler) attempt(w http.ResponseWriter, r *http.Request) {
	auth, ok := h.resolveAuth(w, r)
	if !ok {
		return
	}
	if !auth.IsAuth && auth.GuestHash == "" {
		respondError(w, http.StatusNotFound, "NOT_FOUND", "", "Không tìm thấy đơn hàng")
		return
	}
	viewer := apporder.Viewer{
		GuestHash: auth.GuestHash,
	}
	if auth.IsAuth {
		viewer.CustomerID = auth.User.ID
		viewer.IsAdmin = (auth.User.Role == "admin")
	}
	requestID := r.PathValue("request_id")
	order, err := h.service.Attempt(r.Context(), viewer, requestID)
	if err != nil {
		h.respondError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": toOrderDTO(*order)})
}

func (h orderHandler) list(w http.ResponseWriter, r *http.Request) {
	auth, ok := h.resolveAuth(w, r)
	if !ok {
		return
	}
	if !auth.IsAuth {
		respondError(w, http.StatusUnauthorized, "AUTH_REQUIRED", "", "Vui lòng đăng nhập")
		return
	}
	query := r.URL.Query()
	for k, values := range query {
		if (k != "page" && k != "page_size" && k != "status") || len(values) != 1 {
			respondError(w, http.StatusBadRequest, "INVALID_FIELD", "query", "Tham số truy vấn không hợp lệ")
			return
		}
	}
	page := int64(1)
	if raw := query.Get("page"); raw != "" {
		p, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || p < 1 || p > 100000 {
			respondError(w, http.StatusBadRequest, "INVALID_FIELD", "page", "Trang không hợp lệ")
			return
		}
		page = p
	}
	pageSize := int64(20)
	if raw := query.Get("page_size"); raw != "" {
		ps, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || ps < 1 {
			respondError(w, http.StatusBadRequest, "INVALID_FIELD", "page_size", "Kích thước trang không hợp lệ")
			return
		}
		pageSize = ps
	}

	orders, total, p, ps, err := h.service.List(r.Context(), auth.User.ID, query.Get("status"), page, pageSize)
	if err != nil {
		h.respondError(w, err)
		return
	}
	dtoList := make([]orderDTO, 0, len(orders))
	for _, o := range orders {
		dtoList = append(dtoList, toOrderDTO(o))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"data":      dtoList,
		"total":     total,
		"page":      p,
		"page_size": ps,
	})
}

func (h orderHandler) respondError(w http.ResponseWriter, err error) {
	var ifErr *domainorder.InvalidFieldError
	switch {
	case errors.As(err, &ifErr):
		respondError(w, http.StatusBadRequest, "INVALID_FIELD", ifErr.Field, ifErr.Message)
	case errors.Is(err, apporder.ErrNotFound):
		respondError(w, http.StatusNotFound, "NOT_FOUND", "", "Không tìm thấy đơn hàng")
	case errors.Is(err, apporder.ErrStateConflict):
		respondError(w, http.StatusConflict, "ORDER_STATE_CONFLICT", "", "Đơn chưa ở bước nhận hàng hoặc trạng thái đã thay đổi. Hãy tải lại trang.")
	case errors.Is(err, apporder.ErrIdempotencyConflict):
		respondError(w, http.StatusConflict, "IDEMPOTENCY_CONFLICT", "", "Nội dung yêu cầu khác với lần gửi trước")
	case errors.Is(err, apporder.ErrOutOfStock):
		respondError(w, http.StatusConflict, "OUT_OF_STOCK", "", "Sản phẩm tạm thời hết hàng")
	case errors.Is(err, apporder.ErrQuoteChanged):
		respondError(w, http.StatusConflict, "QUOTE_CHANGED", "", "Giá hoặc thông tin sản phẩm đã thay đổi")
	case errors.Is(err, apporder.ErrQuoteExpired):
		respondError(w, http.StatusConflict, "QUOTE_EXPIRED", "", "Báo giá đã hết hạn, vui lòng tạo báo giá mới")
	case errors.Is(err, apporder.ErrShippingUnavailable):
		h.log.Error().Err(err).Msg("shipping unavailable")
		respondError(w, http.StatusServiceUnavailable, "SHIPPING_UNAVAILABLE", "", "Phương thức vận chuyển tạm thời không khả dụng")
	default:
		h.log.Error().Err(err).Msg("order request failed")
		respondError(w, http.StatusServiceUnavailable, "ORDER_UNAVAILABLE", "", "Dịch vụ đơn hàng tạm thời không khả dụng")
	}
}
