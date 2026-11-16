package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	app "github.com/your-org/aodora/server/internal/application/cart"
)

type cartHandler struct {
	service *app.Service
	guard   *AuthGuard
}

func WithCart(service *app.Service, guard *AuthGuard) RouterOption {
	return func(mux *http.ServeMux) {
		h := cartHandler{service: service, guard: guard}
		origin := authHandler{guard: guard}
		mux.HandleFunc("OPTIONS /api/v1/cart/{path...}", origin.preflight)
		mux.HandleFunc("POST /api/v1/cart/validate", origin.withOrigin(h.validate))
		mux.HandleFunc("OPTIONS /api/v1/me/cart/{path...}", origin.preflight)
		mux.HandleFunc("GET /api/v1/me/cart", h.protected(h.cart))
		mux.HandleFunc("POST /api/v1/me/cart/items", h.protected(h.add))
		mux.HandleFunc("PUT /api/v1/me/cart/items/{product_id}/{sku}", h.protected(h.update))
		mux.HandleFunc("DELETE /api/v1/me/cart/items/{product_id}/{sku}", h.protected(h.remove))
		mux.HandleFunc("POST /api/v1/me/cart/merge", h.protected(h.merge))
	}
}

func (h cartHandler) protected(next http.HandlerFunc) http.HandlerFunc {
	origin := authHandler{guard: h.guard}
	return origin.withOrigin(h.guard.Require(next))
}

func (h cartHandler) cart(w http.ResponseWriter, r *http.Request) {
	user, _ := h.guard.Identity(r)
	data, err := h.service.Cart(r.Context(), user.ID)
	h.respond(w, data, err)
}

func (h cartHandler) add(w http.ResponseWriter, r *http.Request) {
	var input app.Selection
	if !decodeCartJSON(w, r, &input) {
		return
	}
	user, _ := h.guard.Identity(r)
	data, err := h.service.AddItem(r.Context(), user.ID, input)
	h.respond(w, data, err)
}

func (h cartHandler) update(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Quantity        int64  `json:"quantity"`
		Color           string `json:"color"`
		Size            string `json:"size"`
		ProductSlug     string `json:"product_slug"`
		FulfillmentMode string `json:"fulfillment_mode"`
	}
	if !decodeCartJSON(w, r, &input) {
		return
	}
	user, _ := h.guard.Identity(r)
	fulfillmentMode := input.FulfillmentMode
	if fulfillmentMode == "" {
		fulfillmentMode = "ready"
	}
	data, err := h.service.UpdateItem(r.Context(), user.ID, app.Selection{ProductID: r.PathValue("product_id"), SKU: r.PathValue("sku"), Quantity: input.Quantity, Color: input.Color, Size: input.Size, ProductSlug: input.ProductSlug, FulfillmentMode: fulfillmentMode})
	h.respond(w, data, err)
}

func (h cartHandler) remove(w http.ResponseWriter, r *http.Request) {
	user, _ := h.guard.Identity(r)
	data, err := h.service.RemoveItem(r.Context(), user.ID, r.PathValue("product_id"), r.PathValue("sku"))
	h.respond(w, data, err)
}

type mergeRequest struct {
	RequestID string          `json:"request_id"`
	Items     []app.Selection `json:"items"`
}

func (h cartHandler) merge(w http.ResponseWriter, r *http.Request) {
	var input mergeRequest
	if !decodeCartJSON(w, r, &input) {
		return
	}
	user, _ := h.guard.Identity(r)
	data, err := h.service.Merge(r.Context(), user.ID, input.RequestID, input.Items)
	h.respond(w, data, err)
}

func (h cartHandler) validate(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Items []app.Selection `json:"items"`
	}
	if !decodeCartJSON(w, r, &input) {
		return
	}
	data, err := h.service.Validate(r.Context(), input.Items)
	h.respond(w, data, err)
}

func decodeCartJSON(w http.ResponseWriter, r *http.Request, out any) bool {
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

func (h cartHandler) respond(w http.ResponseWriter, data app.Cart, err error) {
	if err != nil {
		var invalid app.Invalid
		switch {
		case errors.As(err, &invalid):
			respondError(w, http.StatusBadRequest, "INVALID_FIELD", invalid.Field, invalid.Message)
		case errors.Is(err, app.ErrConflict):
			respondError(w, http.StatusConflict, "CART_CONFLICT", "", "Giỏ hàng vừa thay đổi; hãy tải lại")
		case errors.Is(err, app.ErrItemUnavailable):
			respondError(w, http.StatusConflict, "CART_ITEM_UNAVAILABLE", "", "Sản phẩm hoặc SKU không còn phù hợp")
		default:
			respondError(w, http.StatusInternalServerError, "CART_UNAVAILABLE", "", "Không thể xử lý giỏ hàng")
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": data})
}
