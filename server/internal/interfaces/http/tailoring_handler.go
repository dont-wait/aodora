package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/rs/zerolog"
	apptailoring "github.com/your-org/aodora/server/internal/application/tailoring"
	domaintailoring "github.com/your-org/aodora/server/internal/domain/tailoring"
)

// TailoringService defines the application operations needed by tailoringHandler.
type TailoringService interface {
	Submit(ctx context.Context, input apptailoring.SubmitInput) (*domaintailoring.TailoringRequest, error)
}

type tailoringHandler struct {
	service TailoringService
	guard   *AuthGuard
	log     zerolog.Logger
}

// WithTailoring registers public tailoring request routes on the router.
func WithTailoring(service TailoringService, guard *AuthGuard, log zerolog.Logger) RouterOption {
	return func(mux *http.ServeMux) {
		h := tailoringHandler{service: service, guard: guard, log: log}
		mux.HandleFunc("OPTIONS /api/v1/tailoring-requests", h.preflight)
		mux.HandleFunc("OPTIONS /api/v1/tailoring-requests/{path...}", h.preflight)
		mux.HandleFunc("POST /api/v1/tailoring-requests", h.wrap(h.create))
	}
}

func (h tailoringHandler) preflight(w http.ResponseWriter, r *http.Request) {
	h.guard.handlePreflight(w, r, nil, "Content-Type, Accept")
}

func (h tailoringHandler) wrap(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		if !h.guard.cors(w, r) {
			respondError(w, http.StatusForbidden, "ORIGIN_FORBIDDEN", "", "Nguồn truy cập không được cấp phép")
			return
		}
		next(w, r)
	}
}

type measurementsDTO struct {
	Body            *string `json:"body"`
	NeckCM          any     `json:"neck_cm"`
	BustCM          any     `json:"bust_cm"`
	WaistCM         any     `json:"waist_cm"`
	HipsCM          any     `json:"hips_cm"`
	ShoulderCM      any     `json:"shoulder_cm"`
	SleeveCM        any     `json:"sleeve_cm"`
	GarmentLengthCM any     `json:"garment_length_cm"`
}

type createTailoringRequestDTO struct {
	FullName         *string          `json:"full_name"`
	Name             *string          `json:"name"`
	Phone            *string          `json:"phone"`
	PreferredContact *string          `json:"preferred_contact"`
	CategoryID       *string          `json:"category_id"`
	ProductID        *string          `json:"product_id"`
	ProductSlug      *string          `json:"product_slug"`
	Measurements     *measurementsDTO `json:"measurements"`
	Notes            *string          `json:"notes"`
	Note             *string          `json:"note"`
}

func (h tailoringHandler) create(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "Content-Type", "Yêu cầu application/json")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	dec := json.NewDecoder(r.Body)
	dec.UseNumber()
	dec.DisallowUnknownFields()

	var req createTailoringRequestDTO
	if err := dec.Decode(&req); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			respondError(w, http.StatusBadRequest, "INVALID_BODY", "body", "Dung lượng yêu cầu vượt quá giới hạn 64 KiB")
			return
		}
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "body", "Dữ liệu JSON không hợp lệ")
		return
	}

	var extra json.RawMessage
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "body", "Dữ liệu JSON thừa")
		return
	}

	user, presented, ok := h.guard.Session(r)
	if presented && !ok {
		respondError(w, http.StatusUnauthorized, "AUTH_REQUIRED", "", "Vui lòng đăng nhập")
		return
	}

	var customerID *string
	if ok {
		customerID = &user.ID
	}

	if req.FullName != nil && req.Name != nil && strings.TrimSpace(*req.FullName) != strings.TrimSpace(*req.Name) {
		respondError(w, http.StatusBadRequest, "INVALID_FIELD", "name", "full_name và name mâu thuẫn; chỉ gửi một trường")
		return
	}
	if req.Notes != nil && req.Note != nil && strings.TrimSpace(*req.Notes) != strings.TrimSpace(*req.Note) {
		respondError(w, http.StatusBadRequest, "INVALID_FIELD", "note", "notes và note mâu thuẫn; chỉ gửi một trường")
		return
	}

	fullName := ""
	if req.FullName != nil {
		fullName = *req.FullName
	} else if req.Name != nil {
		fullName = *req.Name
	}

	phone := ""
	if req.Phone != nil {
		phone = *req.Phone
	}

	var notes *string
	if req.Notes != nil {
		notes = req.Notes
	} else if req.Note != nil {
		notes = req.Note
	}

	var domainMeasurements *domaintailoring.Measurements
	if req.Measurements != nil {
		m := req.Measurements
		neck, err := parseMeasurementNumber(m.NeckCM, "neck_cm")
		if err != nil {
			h.respondError(w, err)
			return
		}
		bust, err := parseMeasurementNumber(m.BustCM, "bust_cm")
		if err != nil {
			h.respondError(w, err)
			return
		}
		waist, err := parseMeasurementNumber(m.WaistCM, "waist_cm")
		if err != nil {
			h.respondError(w, err)
			return
		}
		hips, err := parseMeasurementNumber(m.HipsCM, "hips_cm")
		if err != nil {
			h.respondError(w, err)
			return
		}
		shoulder, err := parseMeasurementNumber(m.ShoulderCM, "shoulder_cm")
		if err != nil {
			h.respondError(w, err)
			return
		}
		sleeve, err := parseMeasurementNumber(m.SleeveCM, "sleeve_cm")
		if err != nil {
			h.respondError(w, err)
			return
		}
		garmentLen, err := parseMeasurementNumber(m.GarmentLengthCM, "garment_length_cm")
		if err != nil {
			h.respondError(w, err)
			return
		}

		domainMeasurements = &domaintailoring.Measurements{
			Body:            m.Body,
			NeckCM:          neck,
			BustCM:          bust,
			WaistCM:         waist,
			HipsCM:          hips,
			ShoulderCM:      shoulder,
			SleeveCM:        sleeve,
			GarmentLengthCM: garmentLen,
		}
	}

	input := apptailoring.SubmitInput{
		CustomerID:       customerID,
		FullName:         fullName,
		Phone:            phone,
		PreferredContact: req.PreferredContact,
		CategoryID:       req.CategoryID,
		ProductID:        req.ProductID,
		ProductSlug:      req.ProductSlug,
		Measurements:     domainMeasurements,
		Notes:            notes,
	}

	res, err := h.service.Submit(r.Context(), input)
	if err != nil {
		h.respondError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"data": map[string]string{
			"id":     res.ID,
			"status": res.Status,
		},
	})
}

func parseMeasurementNumber(val any, fieldName string) (*float64, error) {
	if val == nil {
		return nil, nil
	}
	switch v := val.(type) {
	case json.Number:
		f, err := strconv.ParseFloat(v.String(), 64)
		if err != nil || math.IsNaN(f) || math.IsInf(f, 0) || f <= 0 || f > domaintailoring.MaxMeasurementCM {
			return nil, domaintailoring.NewInvalidFieldError(fieldName, "Số đo phải là số dương hợp lệ")
		}
		return &f, nil
	case float64:
		if math.IsNaN(v) || math.IsInf(v, 0) || v <= 0 || v > domaintailoring.MaxMeasurementCM {
			return nil, domaintailoring.NewInvalidFieldError(fieldName, "Số đo phải là số dương hợp lệ")
		}
		return &v, nil
	default:
		return nil, domaintailoring.NewInvalidFieldError(fieldName, "Số đo phải là số dương hợp lệ")
	}
}

func (h tailoringHandler) respondError(w http.ResponseWriter, err error) {
	var invalid *domaintailoring.InvalidFieldError
	if errors.As(err, &invalid) {
		respondError(w, http.StatusBadRequest, "INVALID_FIELD", invalid.Field, invalid.Message)
		return
	}
	if errors.Is(err, domaintailoring.ErrTailoringUnavailable) {
		respondError(w, http.StatusServiceUnavailable, "TAILORING_UNAVAILABLE", "", "Dịch vụ may đo tạm thời không khả dụng, vui lòng thử lại")
		return
	}
	respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "", "Không thể lưu yêu cầu may đo")
}
