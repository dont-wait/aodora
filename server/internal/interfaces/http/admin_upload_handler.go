package httpapi

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/your-org/aodora/server/internal/application/upload"
)

type adminUploadHandler struct {
	service *upload.Service
	guard   adminCatalogHandler
	log     zerolog.Logger
}

// WithAdminUploads registers POST /api/v1/admin/uploads/images. It reuses the admin guard (admin role,
// origin check, no-store); preflight is served by OPTIONS /api/v1/admin/{path...}.
func WithAdminUploads(service *upload.Service, authorize AdminAuthorizer, log zerolog.Logger, allowedOrigin string) RouterOption {
	return func(mux *http.ServeMux) {
		h := adminUploadHandler{service: service, log: log, guard: adminCatalogHandler{authorize: authorize, log: log, allowedOrigin: strings.TrimSuffix(strings.TrimSpace(allowedOrigin), "/")}}
		mux.HandleFunc("POST /api/v1/admin/uploads/images", h.guard.guard(h.image))
	}
}

func (h adminUploadHandler) image(w http.ResponseWriter, r *http.Request) {
	rc := http.NewResponseController(w)
	_ = rc.SetReadDeadline(time.Now().Add(time.Minute))
	_ = rc.SetWriteDeadline(time.Now().Add(time.Minute))
	r.Body = http.MaxBytesReader(w, r.Body, upload.MaxBytes+(1<<20))
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			respondError(w, http.StatusRequestEntityTooLarge, "FILE_TOO_LARGE", "file", "Ảnh quá lớn, tối đa 10 MB.")
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
	data, err := io.ReadAll(io.LimitReader(file, upload.MaxBytes+1))
	if err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_FIELD", "file", "Không đọc được file.")
		return
	}
	result, err := h.service.Upload(r.Context(), r.FormValue("purpose"), data)
	if err != nil {
		var invalid upload.Invalid
		switch {
		case errors.As(err, &invalid):
			status := http.StatusBadRequest
			if invalid.Field == "file" && strings.Contains(invalid.Message, "tối đa 10 MB") {
				status = http.StatusRequestEntityTooLarge
			}
			respondError(w, status, "INVALID_FIELD", invalid.Field, invalid.Message)
		case errors.Is(err, upload.ErrNotConfigured):
			respondError(w, http.StatusServiceUnavailable, "STORAGE_UNAVAILABLE", "", "Chưa cấu hình kho lưu ảnh (RUSTFS_ACCESS_KEY, RUSTFS_SECRET_KEY, RUSTFS_BUCKET). Liên hệ quản trị hệ thống.")
		default:
			h.log.Error().Str("cause", safeCatalogError(err)).Msg("image upload failed")
			respondError(w, http.StatusServiceUnavailable, "STORAGE_UNAVAILABLE", "", "Không lưu được ảnh vào kho lưu trữ. Vui lòng thử lại sau.")
		}
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"data": result})
}
