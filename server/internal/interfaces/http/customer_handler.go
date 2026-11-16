package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	app "github.com/your-org/aodora/server/internal/application/customer"
)

type customerHandler struct {
	service *app.Service
	guard   *AuthGuard
}

func WithCustomer(service *app.Service, guard *AuthGuard) RouterOption {
	return func(mux *http.ServeMux) {
		h := customerHandler{service: service, guard: guard}
		origin := authHandler{guard: guard}
		mux.HandleFunc("OPTIONS /api/v1/me/{path...}", origin.preflight)
		mux.HandleFunc("GET /api/v1/me/profile", h.protected(h.profile))
		mux.HandleFunc("PATCH /api/v1/me/profile", h.protected(h.updateProfile))
		mux.HandleFunc("GET /api/v1/me/addresses", h.protected(h.addresses))
		mux.HandleFunc("POST /api/v1/me/addresses", h.protected(h.createAddress))
		mux.HandleFunc("PUT /api/v1/me/addresses/{id}", h.protected(h.updateAddress))
		mux.HandleFunc("DELETE /api/v1/me/addresses/{id}", h.protected(h.deleteAddress))
		mux.HandleFunc("GET /api/v1/me/measurements", h.protected(h.measurements))
		mux.HandleFunc("PUT /api/v1/me/measurements", h.protected(h.updateMeasurements))
		mux.HandleFunc("GET /api/v1/me/wishlist", h.protected(h.wishlist))
		mux.HandleFunc("PUT /api/v1/me/wishlist/{product_id}", h.protected(h.addWishlist))
		mux.HandleFunc("DELETE /api/v1/me/wishlist/{product_id}", h.protected(h.removeWishlist))
	}
}

func (h customerHandler) protected(next http.HandlerFunc) http.HandlerFunc {
	origin := authHandler{guard: h.guard}
	return origin.withOrigin(h.guard.Require(next))
}

func (h customerHandler) userID(r *http.Request) (string, bool) {
	user, ok := h.guard.Identity(r)
	return user.ID, ok
}

func (h customerHandler) profile(w http.ResponseWriter, r *http.Request) {
	userID, _ := h.userID(r)
	data, err := h.service.Profile(r.Context(), userID)
	h.respond(w, data, err, http.StatusOK)
}

func (h customerHandler) updateProfile(w http.ResponseWriter, r *http.Request) {
	var input app.ProfileInput
	if !decodeCustomerJSON(w, r, &input) {
		return
	}
	userID, _ := h.userID(r)
	data, err := h.service.UpdateProfile(r.Context(), userID, input)
	h.respond(w, data, err, http.StatusOK)
}

func (h customerHandler) addresses(w http.ResponseWriter, r *http.Request) {
	userID, _ := h.userID(r)
	data, err := h.service.Profile(r.Context(), userID)
	if err != nil {
		h.respond(w, nil, err, 0)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": data.Addresses})
}

func (h customerHandler) createAddress(w http.ResponseWriter, r *http.Request) {
	var input app.AddressInput
	if !decodeCustomerJSON(w, r, &input) {
		return
	}
	userID, _ := h.userID(r)
	data, err := h.service.AddAddress(r.Context(), userID, input)
	h.respond(w, data, err, http.StatusCreated)
}

func (h customerHandler) updateAddress(w http.ResponseWriter, r *http.Request) {
	var input app.AddressInput
	if !decodeCustomerJSON(w, r, &input) {
		return
	}
	userID, _ := h.userID(r)
	data, err := h.service.UpdateAddress(r.Context(), userID, r.PathValue("id"), input)
	h.respond(w, data, err, http.StatusOK)
}

func (h customerHandler) deleteAddress(w http.ResponseWriter, r *http.Request) {
	userID, _ := h.userID(r)
	data, err := h.service.DeleteAddress(r.Context(), userID, r.PathValue("id"))
	h.respond(w, data, err, http.StatusOK)
}

func (h customerHandler) measurements(w http.ResponseWriter, r *http.Request) {
	userID, _ := h.userID(r)
	data, err := h.service.Measurements(r.Context(), userID)
	if err != nil {
		h.respond(w, nil, err, 0)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"measurement_profile": data.Measurements, "measurement_notes": data.MeasurementNotes}})
}

type measurementsInput struct {
	MeasurementProfile *app.Measurements `json:"measurement_profile"`
	MeasurementNotes   string            `json:"measurement_notes"`
}

func (h customerHandler) updateMeasurements(w http.ResponseWriter, r *http.Request) {
	var input measurementsInput
	if !decodeCustomerJSON(w, r, &input) {
		return
	}
	userID, _ := h.userID(r)
	data, err := h.service.UpdateMeasurements(r.Context(), userID, input.MeasurementProfile, input.MeasurementNotes)
	h.respond(w, map[string]any{"measurement_profile": data.Measurements, "measurement_notes": data.MeasurementNotes}, err, http.StatusOK)
}

func (h customerHandler) wishlist(w http.ResponseWriter, r *http.Request) {
	userID, _ := h.userID(r)
	data, err := h.service.Wishlist(r.Context(), userID)
	h.respond(w, map[string]any{"items": data}, err, http.StatusOK)
}

func (h customerHandler) addWishlist(w http.ResponseWriter, r *http.Request) {
	userID, _ := h.userID(r)
	err := h.service.AddWishlist(r.Context(), userID, r.PathValue("product_id"))
	h.respond(w, map[string]any{"saved": true}, err, http.StatusOK)
}

func (h customerHandler) removeWishlist(w http.ResponseWriter, r *http.Request) {
	userID, _ := h.userID(r)
	err := h.service.RemoveWishlist(r.Context(), userID, r.PathValue("product_id"))
	h.respond(w, map[string]any{"saved": false}, err, http.StatusOK)
}

func decodeCustomerJSON(w http.ResponseWriter, r *http.Request, out any) bool {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "Content-Type", "Yêu cầu application/json")
		return false
	}
	decoder := jsonDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "body", "JSON không hợp lệ")
		return false
	}
	return true
}

func jsonDecoder(body io.Reader) *json.Decoder {
	return json.NewDecoder(io.LimitReader(body, 1<<20))
}

func (h customerHandler) respond(w http.ResponseWriter, data any, err error, status int) {
	if err != nil {
		var invalid app.Invalid
		switch {
		case errors.As(err, &invalid):
			respondError(w, http.StatusBadRequest, "INVALID_FIELD", invalid.Field, invalid.Message)
		case errors.Is(err, app.ErrNotFound), errors.Is(err, app.ErrProductNotFound):
			respondError(w, http.StatusNotFound, "NOT_FOUND", "", "Không tìm thấy dữ liệu")
		case errors.Is(err, app.ErrConflict):
			respondError(w, http.StatusConflict, "CONFLICT", "", "Dữ liệu đã thay đổi; hãy tải lại trước khi lưu")
		default:
			respondError(w, http.StatusInternalServerError, "CUSTOMER_UNAVAILABLE", "", "Không thể xử lý dữ liệu tài khoản")
		}
		return
	}
	writeJSON(w, status, map[string]any{"data": data})
}
