package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/your-org/aodora/server/internal/application/auth"
	app "github.com/your-org/aodora/server/internal/application/tailoring"
	domain "github.com/your-org/aodora/server/internal/domain/tailoring"
	"github.com/your-org/aodora/server/internal/logger"
)

type adminTailoringStub struct {
	calls  int
	method string
	id     string
	target string
	filter app.AdminFilter
	input  app.UpdateInput
	item   domain.TailoringRequest
	err    error
}

func (s *adminTailoringStub) rec(m, id, target string) (*domain.TailoringRequest, error) {
	s.calls++
	s.method, s.id, s.target = m, id, target
	if s.err != nil {
		return nil, s.err
	}
	return &s.item, nil
}
func (s *adminTailoringStub) Get(_ context.Context, id string) (*domain.TailoringRequest, error) {
	return s.rec("get", id, "")
}
func (s *adminTailoringStub) Update(_ context.Context, id string, in app.UpdateInput) (*domain.TailoringRequest, error) {
	s.input = in
	return s.rec("update", id, "")
}
func (s *adminTailoringStub) SetStatus(_ context.Context, id, t string) (*domain.TailoringRequest, error) {
	return s.rec("status", id, t)
}
func (s *adminTailoringStub) ConfirmMeasurements(_ context.Context, id string) (*domain.TailoringRequest, error) {
	return s.rec("confirm", id, "")
}
func (s *adminTailoringStub) List(_ context.Context, f app.AdminFilter) ([]domain.TailoringRequest, int64, int64, int64, error) {
	s.calls++
	s.method = "list"
	s.filter = f
	return []domain.TailoringRequest{s.item}, 1, f.Page, f.PageSize, s.err
}

func adminTailoringRouter(log zerolog.Logger) (http.Handler, *adminTailoringStub, *authHTTPRepository, *http.Cookie) {
	token := strings.Repeat("b", 64)
	hash := sha256.Sum256([]byte(token))
	repo := &authHTTPRepository{user: auth.User{ID: "680000000000000000000001", Role: "admin"}, tokenHash: hex.EncodeToString(hash[:]), expiresAt: time.Now().Add(time.Hour)}
	guard := NewAuthGuard(auth.NewService(repo, time.Hour), AuthCookieConfig{Name: "session"})
	stub := &adminTailoringStub{item: domain.TailoringRequest{ID: "6d0000000000000000000001", FullName: "A", Phone: "0900000001", Status: "submitted"}}
	router := NewRouter(nil, log, WithAdminCatalog(nil, guard.RequireAdmin, log, "https://shop.example"), WithAdminTailoring(stub, guard.RequireAdmin, log, "https://shop.example"))
	return router, stub, repo, &http.Cookie{Name: "session", Value: token}
}

func TestAdminTailoringAuthorization(t *testing.T) {
	router, s, repo, cookie := adminTailoringRouter(*logger.NewDiscard())
	id := "/api/v1/admin/tailoring-requests/6d0000000000000000000001"
	routes := []struct{ method, path, body string }{
		{"GET", "/api/v1/admin/tailoring-requests", ""}, {"GET", id, ""}, {"PATCH", id, `{"notes":"x"}`},
		{"PATCH", id + "/status", `{"status":"contacted"}`}, {"POST", id + "/confirm-measurements", `{"confirmed":true}`},
	}
	for _, route := range routes {
		for _, role := range []string{"guest", "customer"} {
			repo.user.Role = role
			c := cookie
			if role == "guest" {
				c = nil
			}
			assertAdminHTTP(t, adminOrderRequest(router, route.method, route.path, route.body, c, nil), 403, "ADMIN_REQUIRED", "")
		}
	}
	if s.calls != 0 {
		t.Fatal("unauthorized call reached service")
	}
	repo.user.Role = "admin"
	assertAdminHTTP(t, adminOrderRequest(router, "GET", routes[0].path, "", cookie, map[string]string{"Origin": "https://evil.example"}), 403, "ORIGIN_FORBIDDEN", "")
	if s.calls != 0 {
		t.Fatal("forbidden origin reached service")
	}
	w := adminOrderRequest(router, "OPTIONS", id+"/status", "", nil, map[string]string{"Origin": "https://shop.example", "Access-Control-Request-Method": "PATCH", "Access-Control-Request-Headers": "Content-Type, Accept"})
	assertAdminHTTP(t, w, 204, "", "")
}

func TestAdminTailoringListQuery(t *testing.T) {
	router, s, _, cookie := adminTailoringRouter(*logger.NewDiscard())
	base := "/api/v1/admin/tailoring-requests"
	for q, field := range map[string]string{"?page=0": "page", "?page_size=abc": "page_size", "?status=": "status", "?status=a&status=b": "query", "?foo=1": "query", "?page=100001": "page"} {
		assertAdminHTTP(t, adminOrderRequest(router, "GET", base+q, "", cookie, nil), 400, "INVALID_FIELD", field)
	}
	if s.calls != 0 {
		t.Fatal("invalid query reached service")
	}
	w := adminOrderRequest(router, "GET", base+"?status=contacted&page=2&page_size=5", "", cookie, nil)
	assertAdminHTTP(t, w, 200, "", "")
	if s.filter.Status != "contacted" || s.filter.Page != 2 || s.filter.PageSize != 5 {
		t.Fatalf("filter=%+v", s.filter)
	}
	var body struct {
		Data  []adminTailoringDTO `json:"data"`
		Total int64               `json:"total"`
		Page  int64               `json:"page"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil || body.Total != 1 || body.Page != 2 || len(body.Data) != 1 || !body.Data[0].IsGuest || len(body.Data[0].AllowedActions) == 0 {
		t.Fatalf("body=%s err=%v", w.Body.String(), err)
	}
}

func TestAdminTailoringPatchDecoding(t *testing.T) {
	router, s, _, cookie := adminTailoringRouter(*logger.NewDiscard())
	path := "/api/v1/admin/tailoring-requests/6d0000000000000000000001"

	for name, body := range map[string]string{
		"status field":   `{"status":"completed"}`,
		"customer_id":    `{"customer_id":"x"}`,
		"updated_at":     `{"updated_at":"2026-01-01T00:00:00Z"}`,
		"unknown nested": `{"measurements":{"chest_cm":1}}`,
		"trailing":       `{"notes":"a"} {}`,
		"array":          `[]`,
	} {
		w := adminOrderRequest(router, "PATCH", path, body, cookie, nil)
		if w.Code != 400 {
			t.Fatalf("%s: expected 400, got %d %s", name, w.Code, w.Body.String())
		}
	}
	for name, c := range map[string]struct{ body, field string }{
		"notes number":    {`{"notes":5}`, "notes"},
		"appointment bad": {`{"appointment_at":"tomorrow"}`, "appointment_at"},
		"appointment num": {`{"appointment_at":5}`, "appointment_at"},
		"measure string":  {`{"measurements":{"waist_cm":"65"}}`, "waist_cm"},
		"measure neg":     {`{"measurements":{"waist_cm":-1}}`, "waist_cm"},
		"measure zero":    {`{"measurements":{"hips_cm":0}}`, "hips_cm"},
		"body number":     {`{"measurements":{"body":1}}`, "body"},
	} {
		assertAdminHTTP(t, adminOrderRequest(router, "PATCH", path, c.body, cookie, nil), 400, "INVALID_FIELD", c.field)
		_ = name
	}
	if s.calls != 0 {
		t.Fatal("invalid body reached service")
	}

	w := adminOrderRequest(router, "PATCH", path, `{"notes":null,"appointment_at":"2026-12-01T09:00:00+07:00","assigned_staff":"Lan","measurements":{"waist_cm":65.5,"bust_cm":null}}`, cookie, nil)
	assertAdminHTTP(t, w, 200, "", "")
	in := s.input
	if !in.Notes.Set || in.Notes.Value != nil {
		t.Fatalf("notes null must be Set+nil: %+v", in.Notes)
	}
	if !in.AppointmentAt.Set || in.AppointmentAt.Value == nil || !in.AppointmentAt.Value.Equal(time.Date(2026, 12, 1, 2, 0, 0, 0, time.UTC)) {
		t.Fatalf("appointment: %+v", in.AppointmentAt)
	}
	if !in.AssignedStaff.Set || *in.AssignedStaff.Value != "Lan" || in.PreferredContact.Set {
		t.Fatalf("staff/contact: %+v", in)
	}
	m := in.Measurements
	if m == nil || !m.WaistCM.Set || *m.WaistCM.Value != 65.5 || !m.BustCM.Set || m.BustCM.Value != nil || m.NeckCM.Set {
		t.Fatalf("measurements: %+v", m)
	}
}

func TestAdminTailoringStatusAndConfirm(t *testing.T) {
	router, s, _, cookie := adminTailoringRouter(*logger.NewDiscard())
	path := "/api/v1/admin/tailoring-requests/6d0000000000000000000001"

	assertAdminHTTP(t, adminOrderRequest(router, "PATCH", path+"/status", `{}`, cookie, nil), 400, "INVALID_FIELD", "status")
	assertAdminHTTP(t, adminOrderRequest(router, "PATCH", path+"/status", `{"status":"contacted","x":1}`, cookie, nil), 400, "INVALID_BODY", "body")
	for _, body := range []string{`{}`, `{"confirmed":false}`, `{"confirmed":"true"}`} {
		w := adminOrderRequest(router, "POST", path+"/confirm-measurements", body, cookie, nil)
		if w.Code != 400 {
			t.Fatalf("%s: expected 400, got %d", body, w.Code)
		}
	}
	if s.calls != 0 {
		t.Fatal("invalid reached service")
	}
	assertAdminHTTP(t, adminOrderRequest(router, "PATCH", path+"/status", `{"status":"contacted"}`, cookie, nil), 200, "", "")
	if s.method != "status" || s.target != "contacted" {
		t.Fatalf("stub=%+v", s)
	}
	assertAdminHTTP(t, adminOrderRequest(router, "POST", path+"/confirm-measurements", `{"confirmed":true}`, cookie, nil), 200, "", "")
	if s.method != "confirm" {
		t.Fatalf("stub=%+v", s)
	}
}

func TestAdminTailoringErrorMapping(t *testing.T) {
	router, s, _, cookie := adminTailoringRouter(*logger.NewDiscard())
	path := "/api/v1/admin/tailoring-requests/6d0000000000000000000001"
	for _, c := range []struct {
		err    error
		status int
		code   string
		field  string
	}{
		{app.ErrNotFound, 404, "NOT_FOUND", ""},
		{app.ErrStateConflict, 409, "TAILORING_STATE_CONFLICT", ""},
		{app.ErrMeasurementLocked, 409, "MEASUREMENT_LOCKED", "measurements"},
		{app.ErrMeasurementIncomplete, 409, "MEASUREMENT_INCOMPLETE", "measurements"},
		{domain.NewInvalidFieldError("status", "x"), 400, "INVALID_FIELD", "status"},
		{domain.ErrTailoringUnavailable, 503, "TAILORING_UNAVAILABLE", ""},
		{errors.New("secret driver text 0900000001"), 503, "TAILORING_UNAVAILABLE", ""},
	} {
		s.err = c.err
		w := adminOrderRequest(router, "GET", path, "", cookie, nil)
		assertAdminHTTP(t, w, c.status, c.code, c.field)
		if strings.Contains(w.Body.String(), "secret") || strings.Contains(w.Body.String(), "0900000001") {
			t.Fatalf("leaked internal text: %s", w.Body.String())
		}
	}
}
