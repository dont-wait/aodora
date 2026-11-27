package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/your-org/aodora/server/internal/application/upload"
)

type memStore struct{ keys []string }

func (m *memStore) Put(_ context.Context, key, _, _ string, body io.Reader, _ int64) error {
	_, _ = io.Copy(io.Discard, body)
	m.keys = append(m.keys, key)
	return nil
}
func (m *memStore) Remove(context.Context, string) error { return nil }
func (m *memStore) PublicURL() string                    { return "http://localhost:9100/aodora-images" }

func uploadMux(allow bool, store *memStore, origin string) *http.ServeMux {
	mux := http.NewServeMux()
	var svc *upload.Service
	if store != nil {
		svc = upload.NewService(store)
	} else {
		svc = upload.NewService(nil)
	}
	WithAdminUploads(svc, func(*http.Request) bool { return allow }, zerolog.Nop(), origin)(mux)
	return mux
}

func pngOf(w, h int) []byte {
	var buf bytes.Buffer
	_ = png.Encode(&buf, image.NewGray(image.Rect(0, 0, w, h)))
	return buf.Bytes()
}

func uploadRequest(t *testing.T, filename, contentType string, data []byte, purpose string) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	header := textproto.MIMEHeader{}
	header.Set("Content-Disposition", `form-data; name="file"; filename="`+filename+`"`)
	header.Set("Content-Type", contentType)
	part, err := w.CreatePart(header)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write(data)
	if purpose != "" {
		_ = w.WriteField("purpose", purpose)
	}
	_ = w.Close()
	r := httptest.NewRequest("POST", "/api/v1/admin/uploads/images", &buf)
	r.Header.Set("Content-Type", w.FormDataContentType())
	return r
}

func do(mux *http.ServeMux, r *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	return w
}

func errorField(t *testing.T, w *httptest.ResponseRecorder) (code, field string) {
	t.Helper()
	var body struct {
		Error struct{ Code, Field string }
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("body %q: %v", w.Body.String(), err)
	}
	return body.Error.Code, body.Error.Field
}

func TestUploadRequiresAdmin(t *testing.T) {
	store := &memStore{}
	w := do(uploadMux(false, store, ""), uploadRequest(t, "a.png", "image/png", pngOf(500, 500), "product"))
	if w.Code != http.StatusForbidden || len(store.keys) != 0 {
		t.Fatalf("status %d, stored %v", w.Code, store.keys)
	}
}

func TestUploadRejectsForeignOrigin(t *testing.T) {
	store := &memStore{}
	r := uploadRequest(t, "a.png", "image/png", pngOf(500, 500), "product")
	r.Header.Set("Origin", "https://evil.example")
	w := do(uploadMux(true, store, "http://localhost:5173"), r)
	if w.Code != http.StatusForbidden || len(store.keys) != 0 {
		t.Fatalf("status %d, stored %v", w.Code, store.keys)
	}
	if code, _ := errorField(t, w); code != "ORIGIN_FORBIDDEN" {
		t.Fatalf("code %s", code)
	}
}

func TestUploadAcceptsValidImageEvenWithWrongNameAndType(t *testing.T) {
	store := &memStore{}
	// A real PNG that claims to be a text file called photo.jpg: the magic bytes decide.
	w := do(uploadMux(true, store, ""), uploadRequest(t, "photo.jpg", "text/plain", pngOf(640, 480), "product"))
	if w.Code != http.StatusCreated {
		t.Fatalf("status %d: %s", w.Code, w.Body.String())
	}
	var body struct {
		Data upload.Result
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(body.Data.Key, ".png") || !strings.HasPrefix(body.Data.Key, "uploads/product/") || body.Data.Width != 640 || body.Data.Height != 480 || body.Data.ContentType != "image/png" {
		t.Fatalf("result %+v", body.Data)
	}
	if !strings.HasPrefix(body.Data.URL, "http://localhost:9100/aodora-images/uploads/product/") || len(store.keys) != 1 {
		t.Fatalf("url %q stored %v", body.Data.URL, store.keys)
	}
}

func TestUploadRejectsWrongContentDespiteImageName(t *testing.T) {
	store := &memStore{}
	w := do(uploadMux(true, store, ""), uploadRequest(t, "photo.jpg", "image/jpeg", []byte("<svg xmlns='http://www.w3.org/2000/svg'></svg>"), "product"))
	if w.Code != http.StatusBadRequest || len(store.keys) != 0 {
		t.Fatalf("status %d stored %v", w.Code, store.keys)
	}
	if code, field := errorField(t, w); code != "INVALID_FIELD" || field != "file" || !strings.Contains(w.Body.String(), "SVG") {
		t.Fatalf("%s %s %s", code, field, w.Body.String())
	}
}

func TestUploadRejectsSmallImagesAndBadPurpose(t *testing.T) {
	store := &memStore{}
	mux := uploadMux(true, store, "")
	w := do(mux, uploadRequest(t, "a.png", "image/png", pngOf(300, 900), "product"))
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "quá nhỏ") {
		t.Fatalf("small: %d %s", w.Code, w.Body.String())
	}
	w = do(mux, uploadRequest(t, "a.png", "image/png", pngOf(500, 500), "banner"))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("purpose: %d", w.Code)
	}
	if _, field := errorField(t, w); field != "purpose" {
		t.Fatalf("field %s", field)
	}
	if len(store.keys) != 0 {
		t.Fatalf("stored %v", store.keys)
	}
}

func TestUploadTooLargeIs413(t *testing.T) {
	store := &memStore{}
	big := append(pngOf(500, 500), make([]byte, upload.MaxBytes+(2<<20))...)
	w := do(uploadMux(true, store, ""), uploadRequest(t, "a.png", "image/png", big, "product"))
	if w.Code != http.StatusRequestEntityTooLarge || len(store.keys) != 0 {
		t.Fatalf("status %d stored %v", w.Code, store.keys)
	}
	// Just over 10 MB but within the multipart overhead allowance: rejected by the service as well.
	over := append(pngOf(500, 500), make([]byte, upload.MaxBytes)...)
	w = do(uploadMux(true, store, ""), uploadRequest(t, "a.png", "image/png", over, "product"))
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("10 MB+ status %d: %s", w.Code, w.Body.String())
	}
}

func TestUploadWithoutStorageIs503(t *testing.T) {
	w := do(uploadMux(true, nil, ""), uploadRequest(t, "a.png", "image/png", pngOf(500, 500), "product"))
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status %d", w.Code)
	}
	if code, _ := errorField(t, w); code != "STORAGE_UNAVAILABLE" {
		t.Fatalf("code %s", code)
	}
	r := httptest.NewRequest("POST", "/api/v1/admin/uploads/images", strings.NewReader("{}"))
	r.Header.Set("Content-Type", "application/json")
	if w := do(uploadMux(true, &memStore{}, ""), r); w.Code != http.StatusBadRequest {
		t.Fatalf("non-multipart: %d", w.Code)
	}
}
