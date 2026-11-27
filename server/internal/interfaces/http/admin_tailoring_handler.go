package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog"
	app "github.com/your-org/aodora/server/internal/application/tailoring"
	domain "github.com/your-org/aodora/server/internal/domain/tailoring"
)

// AdminTailoringService exposes only the operations needed by admin tailoring routes.
type AdminTailoringService interface {
	List(context.Context, app.AdminFilter) ([]domain.TailoringRequest, int64, int64, int64, error)
	Get(context.Context, string) (*domain.TailoringRequest, error)
	Update(context.Context, string, app.UpdateInput) (*domain.TailoringRequest, error)
	SetStatus(context.Context, string, string) (*domain.TailoringRequest, error)
	ConfirmMeasurements(context.Context, string) (*domain.TailoringRequest, error)
}

type adminTailoringHandler struct {
	service AdminTailoringService
	log     zerolog.Logger
}

// WithAdminTailoring registers admin tailoring routes behind the shared admin
// guard. Preflight is already served by OPTIONS /api/v1/admin/{path...}.
func WithAdminTailoring(service AdminTailoringService, authorize AdminAuthorizer, log zerolog.Logger, allowedOrigin string) RouterOption {
	return func(mux *http.ServeMux) {
		h := adminTailoringHandler{service: service, log: log}
		guard := adminCatalogHandler{authorize: authorize, log: log, allowedOrigin: strings.TrimSuffix(strings.TrimSpace(allowedOrigin), "/")}
		for _, route := range []struct {
			pattern string
			handle  http.HandlerFunc
		}{
			{"GET /api/v1/admin/tailoring-requests", h.list},
			{"GET /api/v1/admin/tailoring-requests/{id}", h.get},
			{"PATCH /api/v1/admin/tailoring-requests/{id}", h.update},
			{"PATCH /api/v1/admin/tailoring-requests/{id}/status", h.status},
			{"POST /api/v1/admin/tailoring-requests/{id}/confirm-measurements", h.confirm},
		} {
			mux.HandleFunc(route.pattern, guard.guard(route.handle))
		}
	}
}

type adminMeasurementsDTO struct {
	Body            *string  `json:"body,omitempty"`
	NeckCM          *float64 `json:"neck_cm,omitempty"`
	BustCM          *float64 `json:"bust_cm,omitempty"`
	WaistCM         *float64 `json:"waist_cm,omitempty"`
	HipsCM          *float64 `json:"hips_cm,omitempty"`
	ShoulderCM      *float64 `json:"shoulder_cm,omitempty"`
	SleeveCM        *float64 `json:"sleeve_cm,omitempty"`
	GarmentLengthCM *float64 `json:"garment_length_cm,omitempty"`
}

type adminTailoringDTO struct {
	ID               string                `json:"id"`
	CustomerID       *string               `json:"customer_id"`
	IsGuest          bool                  `json:"is_guest"`
	FullName         string                `json:"full_name"`
	Phone            string                `json:"phone"`
	PreferredContact *string               `json:"preferred_contact"`
	CategoryID       *string               `json:"category_id"`
	ProductID        *string               `json:"product_id"`
	ProductSlug      *string               `json:"product_slug"`
	Measurements     *adminMeasurementsDTO `json:"measurements"`
	Notes            *string               `json:"notes"`
	Status           string                `json:"status"`
	AppointmentAt    *time.Time            `json:"appointment_at"`
	AssignedStaff    *string               `json:"assigned_staff"`
	CreatedAt        time.Time             `json:"created_at"`
	UpdatedAt        time.Time             `json:"updated_at"`
	AllowedActions   []string              `json:"allowed_actions"`
}

func toAdminTailoringDTO(r domain.TailoringRequest) adminTailoringDTO {
	dto := adminTailoringDTO{
		ID: r.ID, CustomerID: r.CustomerID, IsGuest: r.CustomerID == nil, FullName: r.FullName, Phone: r.Phone,
		PreferredContact: r.PreferredContact, CategoryID: r.CategoryID, ProductID: r.ProductID, ProductSlug: r.ProductSlug,
		Notes: r.Notes, Status: r.Status, AppointmentAt: r.AppointmentAt, AssignedStaff: r.AssignedStaff,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt, AllowedActions: domain.AllowedActions(r),
	}
	if m := r.Measurements; m != nil {
		dto.Measurements = &adminMeasurementsDTO{Body: m.Body, NeckCM: m.NeckCM, BustCM: m.BustCM, WaistCM: m.WaistCM,
			HipsCM: m.HipsCM, ShoulderCM: m.ShoulderCM, SleeveCM: m.SleeveCM, GarmentLengthCM: m.GarmentLengthCM}
	}
	return dto
}

// patchField records whether a JSON key was present and its raw value, so
// absent (unchanged) can be told apart from null (clear).
type patchField struct {
	Set bool
	Raw json.RawMessage
}

func (p *patchField) UnmarshalJSON(b []byte) error {
	p.Set = true
	p.Raw = append(p.Raw[:0], b...)
	return nil
}

func (p patchField) isNull() bool { return bytes.Equal(bytes.TrimSpace(p.Raw), []byte("null")) }

func (p patchField) text(field string) (app.Optional[string], error) {
	if !p.Set {
		return app.Optional[string]{}, nil
	}
	if p.isNull() {
		return app.Optional[string]{Set: true}, nil
	}
	var s string
	if err := json.Unmarshal(p.Raw, &s); err != nil {
		return app.Optional[string]{}, domain.NewInvalidFieldError(field, "Phải là chuỗi hoặc null")
	}
	return app.Optional[string]{Set: true, Value: &s}, nil
}

func (p patchField) number(field string) (app.Optional[float64], error) {
	if !p.Set {
		return app.Optional[float64]{}, nil
	}
	if p.isNull() {
		return app.Optional[float64]{Set: true}, nil
	}
	dec := json.NewDecoder(bytes.NewReader(p.Raw))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return app.Optional[float64]{}, domain.NewInvalidFieldError(field, "Số đo phải là số dương hợp lệ")
	}
	f, err := parseMeasurementNumber(v, field)
	if err != nil {
		return app.Optional[float64]{}, err
	}
	return app.Optional[float64]{Set: true, Value: f}, nil
}

type measurementsPatchDTO struct {
	Body            patchField `json:"body"`
	NeckCM          patchField `json:"neck_cm"`
	BustCM          patchField `json:"bust_cm"`
	WaistCM         patchField `json:"waist_cm"`
	HipsCM          patchField `json:"hips_cm"`
	ShoulderCM      patchField `json:"shoulder_cm"`
	SleeveCM        patchField `json:"sleeve_cm"`
	GarmentLengthCM patchField `json:"garment_length_cm"`
}

type updateTailoringDTO struct {
	Notes            patchField            `json:"notes"`
	AppointmentAt    patchField            `json:"appointment_at"`
	AssignedStaff    patchField            `json:"assigned_staff"`
	PreferredContact patchField            `json:"preferred_contact"`
	Measurements     *measurementsPatchDTO `json:"measurements"`
}

func (d updateTailoringDTO) toInput() (app.UpdateInput, error) {
	var in app.UpdateInput
	var err error
	if in.Notes, err = d.Notes.text("notes"); err != nil {
		return in, err
	}
	if in.AssignedStaff, err = d.AssignedStaff.text("assigned_staff"); err != nil {
		return in, err
	}
	if in.PreferredContact, err = d.PreferredContact.text("preferred_contact"); err != nil {
		return in, err
	}
	if d.AppointmentAt.Set {
		in.AppointmentAt.Set = true
		if !d.AppointmentAt.isNull() {
			var raw string
			if err := json.Unmarshal(d.AppointmentAt.Raw, &raw); err != nil {
				return in, domain.NewInvalidFieldError("appointment_at", "Phải là chuỗi RFC3339 hoặc null")
			}
			t, err := time.Parse(time.RFC3339, raw)
			if err != nil {
				return in, domain.NewInvalidFieldError("appointment_at", "Phải là thời điểm RFC3339 hợp lệ")
			}
			in.AppointmentAt.Value = &t
		}
	}
	if m := d.Measurements; m != nil {
		p := &app.MeasurementsPatch{}
		if p.Body, err = m.Body.text("body"); err != nil {
			return in, err
		}
		for _, f := range []struct {
			src  patchField
			name string
			dst  *app.Optional[float64]
		}{
			{m.NeckCM, "neck_cm", &p.NeckCM}, {m.BustCM, "bust_cm", &p.BustCM}, {m.WaistCM, "waist_cm", &p.WaistCM},
			{m.HipsCM, "hips_cm", &p.HipsCM}, {m.ShoulderCM, "shoulder_cm", &p.ShoulderCM},
			{m.SleeveCM, "sleeve_cm", &p.SleeveCM}, {m.GarmentLengthCM, "garment_length_cm", &p.GarmentLengthCM},
		} {
			if *f.dst, err = f.src.number(f.name); err != nil {
				return in, err
			}
		}
		in.Measurements = p
	}
	return in, nil
}

func (h adminTailoringHandler) list(w http.ResponseWriter, r *http.Request) {
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
			if f.Status == "" {
				respondError(w, 400, "INVALID_FIELD", "status", "Trạng thái yêu cầu may đo không hợp lệ")
				return
			}
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
	items, total, page, size, err := h.service.List(r.Context(), f)
	if err != nil {
		h.respondError(w, err)
		return
	}
	data := make([]adminTailoringDTO, len(items))
	for i, it := range items {
		data[i] = toAdminTailoringDTO(it)
	}
	writeJSON(w, 200, map[string]any{"data": data, "total": total, "page": page, "page_size": size})
}

func (h adminTailoringHandler) get(w http.ResponseWriter, r *http.Request) {
	t, err := h.service.Get(r.Context(), r.PathValue("id"))
	h.respond(w, t, err)
}

func (h adminTailoringHandler) update(w http.ResponseWriter, r *http.Request) {
	var body updateTailoringDTO
	if !decodeAdminOrderBody(w, r, &body, false) {
		return
	}
	in, err := body.toInput()
	if err != nil {
		h.respondError(w, err)
		return
	}
	t, err := h.service.Update(r.Context(), r.PathValue("id"), in)
	h.respond(w, t, err)
}

func (h adminTailoringHandler) status(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Status *string `json:"status"`
	}
	if !decodeAdminOrderBody(w, r, &body, false) {
		return
	}
	if body.Status == nil {
		h.respondError(w, domain.NewInvalidFieldError("status", "status là bắt buộc"))
		return
	}
	t, err := h.service.SetStatus(r.Context(), r.PathValue("id"), *body.Status)
	h.respond(w, t, err)
}

func (h adminTailoringHandler) confirm(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Confirmed *bool `json:"confirmed"`
	}
	if !decodeAdminOrderBody(w, r, &body, false) {
		return
	}
	if body.Confirmed == nil || !*body.Confirmed {
		h.respondError(w, domain.NewInvalidFieldError("confirmed", "Phải gửi confirmed=true để xác nhận số đo"))
		return
	}
	t, err := h.service.ConfirmMeasurements(r.Context(), r.PathValue("id"))
	h.respond(w, t, err)
}

func (h adminTailoringHandler) respond(w http.ResponseWriter, t *domain.TailoringRequest, err error) {
	if err != nil {
		h.respondError(w, err)
		return
	}
	if t == nil {
		h.respondError(w, domain.ErrTailoringUnavailable)
		return
	}
	writeJSON(w, 200, map[string]any{"data": toAdminTailoringDTO(*t)})
}

func (h adminTailoringHandler) respondError(w http.ResponseWriter, err error) {
	var invalid *domain.InvalidFieldError
	switch {
	case errors.As(err, &invalid):
		respondError(w, 400, "INVALID_FIELD", invalid.Field, invalid.Message)
	case errors.Is(err, app.ErrNotFound):
		respondError(w, 404, "NOT_FOUND", "", "Không tìm thấy yêu cầu may đo")
	case errors.Is(err, app.ErrStateConflict):
		respondError(w, 409, "TAILORING_STATE_CONFLICT", "", "Trạng thái yêu cầu không cho phép thao tác này")
	case errors.Is(err, app.ErrMeasurementLocked):
		respondError(w, 409, "MEASUREMENT_LOCKED", "measurements", "Số đo đã chốt hoặc yêu cầu đã kết thúc, không thể sửa")
	case errors.Is(err, app.ErrMeasurementIncomplete):
		respondError(w, 409, "MEASUREMENT_INCOMPLETE", "measurements", "Cần đủ bảy số đo (cm) trước khi xác nhận")
	default:
		// Never log raw driver text, body, token or phone number.
		h.log.Error().Str("error_type", fmt.Sprintf("%T", err)).Msg("admin tailoring request failed")
		respondError(w, 503, "TAILORING_UNAVAILABLE", "", "Dịch vụ may đo tạm thời không khả dụng")
	}
}
