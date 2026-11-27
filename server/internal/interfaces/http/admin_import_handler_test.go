package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	app "github.com/your-org/aodora/server/internal/application/catalogadmin"
	dashboardapp "github.com/your-org/aodora/server/internal/application/dashboard"
)

type importRepo struct {
	app.Repository
	created int
}

func (r *importRepo) Categories(context.Context) ([]map[string]any, error) {
	return []map[string]any{{"id": "670000000000000000000001", "slug": "ao-nam", "name": "Áo nam", "parent_id": "", "is_active": true, "sort_order": int64(1), "version": int64(0)}}, nil
}
func (r *importRepo) CreateCategory(_ context.Context, v app.CategoryInput) (map[string]any, error) {
	r.created++
	return map[string]any{"id": "670000000000000000000002", "slug": v.Slug}, nil
}
func (r *importRepo) Products(context.Context, int64, int64) (app.ProductPage, error) {
	return app.ProductPage{TotalPages: 1}, nil
}

func importMux(t *testing.T, allow bool, repo *importRepo) *http.ServeMux {
	t.Helper()
	mux := http.NewServeMux()
	WithAdminImport(app.NewService(repo), func(*http.Request) bool { return allow }, zerolog.Nop(), "")(mux)
	return mux
}

func multipartBody(t *testing.T, filename, content string) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile("file", filename)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write([]byte(content))
	_ = w.Close()
	return &buf, w.FormDataContentType()
}

func TestAdminImportRequiresAdmin(t *testing.T) {
	mux := importMux(t, false, &importRepo{})
	for _, req := range []*http.Request{
		httptest.NewRequest("POST", "/api/v1/admin/import/categories", strings.NewReader("x")),
		httptest.NewRequest("GET", "/api/v1/admin/export/products", nil),
	} {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Fatalf("%s %s: status %d", req.Method, req.URL.Path, w.Code)
		}
	}
}

func TestAdminImportDryRunThenCommit(t *testing.T) {
	repo := &importRepo{}
	mux := importMux(t, true, repo)
	csvText := "\xef\xbb\xbfname,slug,description,parent_slug,sort_order,is_active,image_url,image_alt\nÁo nữ,ao-nu,,,2,true,,\n"
	post := func(mode string) map[string]any {
		body, contentType := multipartBody(t, "categories.csv", csvText)
		r := httptest.NewRequest("POST", "/api/v1/admin/import/categories?mode="+mode, body)
		r.Header.Set("Content-Type", contentType)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("%s: status %d %s", mode, w.Code, w.Body.String())
		}
		var out struct{ Data map[string]any }
		if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
			t.Fatal(err)
		}
		return out.Data
	}
	dry := post("dry_run")
	if dry["created"] != float64(1) || dry["committed"] != false || repo.created != 0 {
		t.Fatalf("dry run: %v created=%d", dry, repo.created)
	}
	done := post("commit")
	if done["committed"] != true || repo.created != 1 {
		t.Fatalf("commit: %v created=%d", done, repo.created)
	}
}

func TestAdminImportReportsFileLevelErrorsAndRejectsBadInput(t *testing.T) {
	mux := importMux(t, true, &importRepo{})
	body, contentType := multipartBody(t, "x.csv", "name,description\nA,B\n")
	r := httptest.NewRequest("POST", "/api/v1/admin/import/categories", body)
	r.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "Thiếu cột bắt buộc") {
		t.Fatalf("missing column must be reported per row: %d %s", w.Code, w.Body.String())
	}
	for name, req := range map[string]*http.Request{
		"bad mode": httptest.NewRequest("POST", "/api/v1/admin/import/categories?mode=oops", strings.NewReader("x")),
		"bad kind": httptest.NewRequest("POST", "/api/v1/admin/import/orders", strings.NewReader("x")),
		"bad type": httptest.NewRequest("POST", "/api/v1/admin/import/categories", strings.NewReader("x")),
	} {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code < 400 {
			t.Errorf("%s: status %d", name, w.Code)
		}
	}
}

func TestAdminExportIsCSVWithBOM(t *testing.T) {
	mux := importMux(t, true, &importRepo{})
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/admin/export/categories", nil))
	if w.Code != http.StatusOK || !strings.HasPrefix(w.Header().Get("Content-Type"), "text/csv") || !strings.Contains(w.Header().Get("Content-Disposition"), "aodora-categories.csv") {
		t.Fatalf("headers: %d %v", w.Code, w.Header())
	}
	body := w.Body.String()
	if !strings.HasPrefix(body, "\xef\xbb\xbf") || !strings.Contains(body, "name,slug,description,parent_slug") || !strings.Contains(body, "Áo nam,ao-nam") {
		t.Fatalf("body: %q", body)
	}
}

type failingDashboard struct{}

func (failingDashboard) Report(context.Context, string, string, string, int64) (dashboardapp.Report, error) {
	return dashboardapp.Report{}, dashboardapp.Invalid{Field: "from", Message: "x"}
}

func TestAdminDashboardValidatesQueryBeforeService(t *testing.T) {
	mux := http.NewServeMux()
	WithAdminDashboard(&failingDashboard{}, func(*http.Request) bool { return true }, zerolog.Nop(), "")(mux)
	for _, path := range []string{"/api/v1/admin/dashboard?low_stock=abc", "/api/v1/admin/dashboard?evil=1", "/api/v1/admin/dashboard?from=a&from=b"} {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if w.Code != http.StatusBadRequest {
			t.Errorf("%s: status %d", path, w.Code)
		}
	}
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/admin/dashboard", nil))
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "INVALID_QUERY") {
		t.Fatalf("service Invalid must map to 400: %d %s", w.Code, w.Body.String())
	}
	deny := http.NewServeMux()
	WithAdminDashboard(&failingDashboard{}, func(*http.Request) bool { return false }, zerolog.Nop(), "")(deny)
	w = httptest.NewRecorder()
	deny.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/admin/dashboard", nil))
	if w.Code != http.StatusForbidden {
		t.Fatalf("non-admin must be forbidden: %d", w.Code)
	}
}
