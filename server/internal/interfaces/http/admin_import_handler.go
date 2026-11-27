package httpapi

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog"
	app "github.com/your-org/aodora/server/internal/application/catalogadmin"
)

type adminImportHandler struct {
	service *app.Service
	guard   adminCatalogHandler
	log     zerolog.Logger
}

// WithAdminImport registers bulk import/export of categories and products.
// It reuses the catalog guard; its OPTIONS /api/v1/admin/{path...} serves preflight.
func WithAdminImport(service *app.Service, authorize AdminAuthorizer, log zerolog.Logger, allowedOrigin string) RouterOption {
	return func(mux *http.ServeMux) {
		h := adminImportHandler{service: service, log: log, guard: adminCatalogHandler{authorize: authorize, log: log, allowedOrigin: strings.TrimSuffix(strings.TrimSpace(allowedOrigin), "/")}}
		mux.HandleFunc("POST /api/v1/admin/import/{kind}", h.guard.guard(h.importFile))
		mux.HandleFunc("GET /api/v1/admin/export/{kind}", h.guard.guard(h.export))
	}
}

func (h adminImportHandler) importFile(w http.ResponseWriter, r *http.Request) {
	kind := r.PathValue("kind")
	if kind != "categories" && kind != "products" {
		respondError(w, http.StatusNotFound, "NOT_FOUND", "", "Loại dữ liệu không hợp lệ")
		return
	}
	commit := false
	switch r.URL.Query().Get("mode") {
	case "", "dry_run":
	case "commit":
		commit = true
	default:
		respondError(w, http.StatusBadRequest, "INVALID_QUERY", "mode", "mode phải là dry_run hoặc commit")
		return
	}
	// A commit writes row by row; give it more time than the default 10s server write timeout.
	rc := http.NewResponseController(w)
	_ = rc.SetWriteDeadline(time.Now().Add(2 * time.Minute))
	_ = rc.SetReadDeadline(time.Now().Add(time.Minute))

	data, format, err := readImportBody(w, r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_FIELD", "file", err.Error())
		return
	}
	allowed, required := app.CategoryColumns(), []string{"name", "slug"}
	if kind == "products" {
		allowed, required = app.ProductColumns(), []string{"product_slug", "name", "category_slug", "price", "sku"}
	}
	rows, parseErrs := app.ParseRows(data, format, allowed, required)
	if len(parseErrs) > 0 {
		writeJSON(w, http.StatusOK, map[string]any{"data": app.ImportReport{Mode: modeOf(commit), Errors: parseErrs, Notes: []string{}, Items: []app.ImportItem{}}})
		return
	}
	var report app.ImportReport
	if kind == "categories" {
		report, err = h.service.ImportCategories(r.Context(), rows, commit)
	} else {
		report, err = h.service.ImportProducts(r.Context(), rows, commit)
	}
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			respondError(w, http.StatusGatewayTimeout, "ADMIN_TIMEOUT", "", "Nhập dữ liệu quá thời gian; kiểm tra lại danh sách rồi chạy lại")
			return
		}
		h.log.Error().Str("kind", kind).Str("cause", safeCatalogError(err)).Msg("admin import failed")
		respondError(w, http.StatusInternalServerError, "ADMIN_CATALOG_UNAVAILABLE", "", "Không thể nhập dữ liệu")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": report})
}

func modeOf(commit bool) string {
	if commit {
		return "commit"
	}
	return "dry_run"
}

// readImportBody accepts multipart (field "file") or a raw text/csv / application/json body.
func readImportBody(w http.ResponseWriter, r *http.Request) ([]byte, string, error) {
	limit := int64(app.MaxImportSize) + (1 << 20)
	r.Body = http.MaxBytesReader(w, r.Body, limit)
	contentType, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	switch contentType {
	case "multipart/form-data":
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			return nil, "", errors.New("File quá lớn hoặc không đọc được")
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			return nil, "", errors.New("Thiếu trường file")
		}
		defer file.Close()
		data, err := io.ReadAll(io.LimitReader(file, int64(app.MaxImportSize)+1))
		if err != nil {
			return nil, "", errors.New("Không đọc được file")
		}
		format := "csv"
		if strings.EqualFold(filepath.Ext(header.Filename), ".json") || strings.Contains(header.Header.Get("Content-Type"), "json") {
			format = "json"
		}
		return data, format, nil
	case "application/json":
		data, err := io.ReadAll(io.LimitReader(r.Body, int64(app.MaxImportSize)+1))
		if err != nil {
			return nil, "", errors.New("Không đọc được nội dung")
		}
		return data, "json", nil
	case "text/csv", "text/plain", "application/csv":
		data, err := io.ReadAll(io.LimitReader(r.Body, int64(app.MaxImportSize)+1))
		if err != nil {
			return nil, "", errors.New("Không đọc được nội dung")
		}
		return data, "csv", nil
	}
	return nil, "", errors.New("Content-Type phải là multipart/form-data, text/csv hoặc application/json")
}

func (h adminImportHandler) export(w http.ResponseWriter, r *http.Request) {
	var rows [][]string
	var err error
	kind := r.PathValue("kind")
	switch kind {
	case "categories":
		rows, err = h.service.ExportCategories(r.Context())
	case "products":
		rows, err = h.service.ExportProducts(r.Context())
	default:
		respondError(w, http.StatusNotFound, "NOT_FOUND", "", "Loại dữ liệu không hợp lệ")
		return
	}
	if err != nil {
		h.log.Error().Str("kind", kind).Str("cause", safeCatalogError(err)).Msg("admin export failed")
		respondError(w, http.StatusInternalServerError, "ADMIN_CATALOG_UNAVAILABLE", "", "Không thể xuất dữ liệu")
		return
	}
	var buf bytes.Buffer
	buf.WriteString("\xef\xbb\xbf") // UTF-8 BOM so Excel opens Vietnamese text correctly
	writer := csv.NewWriter(&buf)
	_ = writer.WriteAll(rows)
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="aodora-`+kind+`.csv"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(buf.Bytes())
}
