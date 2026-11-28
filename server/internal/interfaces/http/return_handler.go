package httpapi

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog"
	returnsapp "github.com/your-org/aodora/server/internal/application/returns"
	"github.com/your-org/aodora/server/internal/application/upload"
	dr "github.com/your-org/aodora/server/internal/domain/returns"
)

type ReturnService interface {
	Eligibility(ctx context.Context, customerID, orderID string) (returnsapp.Eligibility, error)
	Create(ctx context.Context, customerID string, in returnsapp.Input) (dr.Request, error)
	ListOwn(ctx context.Context, customerID string) ([]dr.Request, error)
	Cancel(ctx context.Context, customerID, id string) (*dr.Request, error)
}

type AdminReturnService interface {
	AdminList(ctx context.Context, f returnsapp.Filter) ([]dr.Request, int64, error)
	AdminGet(ctx context.Context, id string) (*dr.Request, error)
	AdminTransition(ctx context.Context, id string, in returnsapp.TransitionInput) (*dr.Request, error)
}

type EvidenceUploader interface {
	UploadEvidence(ctx context.Context, customerID string, data []byte) (upload.EvidenceResult, error)
}

type returnHandler struct {
	service  ReturnService
	uploader EvidenceUploader
	guard    *AuthGuard
	log      zerolog.Logger
}

// WithReturns registers the customer return-request routes (login required).
func WithReturns(service ReturnService, uploader EvidenceUploader, guard *AuthGuard, log zerolog.Logger) RouterOption {
	return func(mux *http.ServeMux) {
		h := returnHandler{service: service, uploader: uploader, guard: guard, log: log}
		origin := authHandler{guard: guard}
		protected := func(next http.HandlerFunc) http.HandlerFunc { return origin.withOrigin(guard.Require(next)) }
		mux.HandleFunc("OPTIONS /api/v1/me/returns", origin.preflight)
		mux.HandleFunc("OPTIONS /api/v1/me/returns/{path...}", origin.preflight)
		mux.HandleFunc("GET /api/v1/me/returns", protected(h.list))
		mux.HandleFunc("POST /api/v1/me/returns", protected(h.create))
		mux.HandleFunc("GET /api/v1/me/returns/eligibility", protected(h.eligibility))
		mux.HandleFunc("POST /api/v1/me/returns/evidence", protected(h.evidence))
		mux.HandleFunc("POST /api/v1/me/returns/{id}/cancel", protected(h.cancel))
	}
}

// WithAdminReturns registers return review routes behind the admin guard.
func WithAdminReturns(service AdminReturnService, authorize AdminAuthorizer, log zerolog.Logger, allowedOrigin string) RouterOption {
	return func(mux *http.ServeMux) {
		h := adminReturnHandler{service: service, log: log}
		admin := adminCatalogHandler{authorize: authorize, log: log, allowedOrigin: strings.TrimSuffix(strings.TrimSpace(allowedOrigin), "/")}
		mux.HandleFunc("GET /api/v1/admin/returns", admin.guard(h.list))
		mux.HandleFunc("GET /api/v1/admin/returns/{id}", admin.guard(h.get))
		mux.HandleFunc("POST /api/v1/admin/returns/{id}/transition", admin.guard(h.transition))
	}
}

func (h returnHandler) userID(r *http.Request) string {
	user, _ := h.guard.Identity(r)
	return user.ID
}

func (h returnHandler) list(w http.ResponseWriter, r *http.Request) {
	data, err := h.service.ListOwn(r.Context(), h.userID(r))
	respondReturn(w, h.log, data, err)
}

func (h returnHandler) eligibility(w http.ResponseWriter, r *http.Request) {
	orderID := r.URL.Query().Get("order_id")
	if orderID == "" {
		respondError(w, http.StatusBadRequest, "INVALID_FIELD", "order_id", "Thiếu mã đơn hàng")
		return
	}
	data, err := h.service.Eligibility(r.Context(), h.userID(r), orderID)
	respondReturn(w, h.log, data, err)
}

func (h returnHandler) create(w http.ResponseWriter, r *http.Request) {
	var in returnsapp.Input
	if !decodeReviewJSON(w, r, &in) {
		return
	}
	data, err := h.service.Create(r.Context(), h.userID(r), in)
	if err == nil {
		writeJSON(w, http.StatusCreated, map[string]any{"data": data})
		return
	}
	respondReturn(w, h.log, nil, err)
}

func (h returnHandler) cancel(w http.ResponseWriter, r *http.Request) {
	data, err := h.service.Cancel(r.Context(), h.userID(r), r.PathValue("id"))
	respondReturn(w, h.log, data, err)
}

func (h returnHandler) evidence(w http.ResponseWriter, r *http.Request) {
	rc := http.NewResponseController(w)
	_ = rc.SetReadDeadline(time.Now().Add(3 * time.Minute))
	_ = rc.SetWriteDeadline(time.Now().Add(3 * time.Minute))
	r.Body = http.MaxBytesReader(w, r.Body, upload.MaxEvidenceVideoBytes+(1<<20))
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
			respondError(w, http.StatusRequestEntityTooLarge, "FILE_TOO_LARGE", "file", "File quá lớn: ảnh tối đa 10 MB, video tối đa 50 MB.")
			return
		}
		respondError(w, http.StatusBadRequest, "INVALID_FIELD", "file", "Yêu cầu phải là multipart/form-data có trường file.")
		return
	}
	defer func() {
		if r.MultipartForm != nil {
			_ = r.MultipartForm.RemoveAll()
		}
	}()
	file, _, err := r.FormFile("file")
	if err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_FIELD", "file", "Thiếu trường file.")
		return
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, upload.MaxEvidenceVideoBytes+1))
	if err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_FIELD", "file", "Không đọc được file.")
		return
	}
	result, err := h.uploader.UploadEvidence(r.Context(), h.userID(r), data)
	if err != nil {
		var invalid upload.Invalid
		switch {
		case errors.As(err, &invalid):
			status := http.StatusBadRequest
			if strings.Contains(invalid.Message, "tối đa") {
				status = http.StatusRequestEntityTooLarge
			}
			respondError(w, status, "INVALID_FIELD", invalid.Field, invalid.Message)
		case errors.Is(err, upload.ErrNotConfigured):
			respondError(w, http.StatusServiceUnavailable, "STORAGE_UNAVAILABLE", "", "Hệ thống chưa nhận được file minh chứng lúc này. Vui lòng thử lại sau hoặc liên hệ cửa hàng.")
		default:
			h.log.Error().Str("cause", safeCatalogError(err)).Msg("return evidence upload failed")
			respondError(w, http.StatusServiceUnavailable, "STORAGE_UNAVAILABLE", "", "Không lưu được file minh chứng. Vui lòng thử lại sau.")
		}
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"data": result})
}

type adminReturnHandler struct {
	service AdminReturnService
	log     zerolog.Logger
}

type adminReturnDTO struct {
	dr.Request
	AllowedNext []string `json:"allowed_next"`
}

func toAdminReturn(r dr.Request) adminReturnDTO {
	return adminReturnDTO{Request: r, AllowedNext: dr.AdminNext(r.Status)}
}

func (h adminReturnHandler) list(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := returnsapp.Filter{Status: q.Get("status"), OrderID: q.Get("order_id")}
	f.Page, _ = strconv.ParseInt(q.Get("page"), 10, 64)
	f.Size, _ = strconv.ParseInt(q.Get("size"), 10, 64)
	items, total, err := h.service.AdminList(r.Context(), f)
	if err != nil {
		respondReturn(w, h.log, nil, err)
		return
	}
	out := make([]adminReturnDTO, len(items))
	for i, it := range items {
		out[i] = toAdminReturn(it)
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": out, "meta": map[string]int64{"total": total}})
}

func (h adminReturnHandler) get(w http.ResponseWriter, r *http.Request) {
	data, err := h.service.AdminGet(r.Context(), r.PathValue("id"))
	if err != nil {
		respondReturn(w, h.log, nil, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": toAdminReturn(*data)})
}

func (h adminReturnHandler) transition(w http.ResponseWriter, r *http.Request) {
	var in returnsapp.TransitionInput
	if !decodeReviewJSON(w, r, &in) {
		return
	}
	data, err := h.service.AdminTransition(r.Context(), r.PathValue("id"), in)
	if err != nil {
		respondReturn(w, h.log, nil, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": toAdminReturn(*data)})
}

func respondReturn(w http.ResponseWriter, log zerolog.Logger, data any, err error) {
	if err == nil {
		writeJSON(w, http.StatusOK, map[string]any{"data": data})
		return
	}
	var invalid returnsapp.Invalid
	var notEligible returnsapp.NotEligible
	switch {
	case errors.As(err, &invalid):
		respondError(w, http.StatusBadRequest, "INVALID_FIELD", invalid.Field, invalid.Message)
	case errors.As(err, &notEligible):
		respondError(w, http.StatusUnprocessableEntity, "RETURN_NOT_ELIGIBLE", "", notEligible.Message)
	case errors.Is(err, returnsapp.ErrNotFound):
		respondError(w, http.StatusNotFound, "NOT_FOUND", "", "Không tìm thấy đơn hàng hoặc yêu cầu hoàn hàng")
	case errors.Is(err, returnsapp.ErrConflict):
		respondError(w, http.StatusConflict, "STATE_CONFLICT", "", "Yêu cầu đã thay đổi hoặc không thể chuyển sang trạng thái này. Hãy tải lại trang.")
	default:
		log.Error().Err(err).Msg("return request failed")
		respondError(w, http.StatusServiceUnavailable, "RETURN_UNAVAILABLE", "", "Dịch vụ hoàn hàng tạm thời không khả dụng")
	}
}
