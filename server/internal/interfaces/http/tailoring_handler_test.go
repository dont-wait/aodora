package httpapi

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
	appauth "github.com/your-org/aodora/server/internal/application/auth"
	apptailoring "github.com/your-org/aodora/server/internal/application/tailoring"
	domaintailoring "github.com/your-org/aodora/server/internal/domain/tailoring"
)

type mockTailoringService struct {
	lastInput *apptailoring.SubmitInput
	errToRet  error
	reqToRet  *domaintailoring.TailoringRequest
}

func (m *mockTailoringService) Submit(_ context.Context, input apptailoring.SubmitInput) (*domaintailoring.TailoringRequest, error) {
	m.lastInput = &input
	if m.errToRet != nil {
		return nil, m.errToRet
	}
	if m.reqToRet != nil {
		return m.reqToRet, nil
	}
	return &domaintailoring.TailoringRequest{
		ID:         "6d0000000000000000000001",
		CustomerID: input.CustomerID,
		FullName:   input.FullName,
		Phone:      input.Phone,
		Status:     domaintailoring.StatusSubmitted,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}, nil
}

type tailoringAuthRepo struct {
	users map[string]appauth.User
}

func newTailoringAuthRepo() *tailoringAuthRepo {
	return &tailoringAuthRepo{users: make(map[string]appauth.User)}
}

func (r *tailoringAuthRepo) addSession(token string, user appauth.User) {
	digest := sha256.Sum256([]byte(token))
	r.users[hex.EncodeToString(digest[:])] = user
}

func (r *tailoringAuthRepo) CreateCustomer(context.Context, appauth.RegisterInput, string) (appauth.User, error) {
	return appauth.User{}, nil
}

func (r *tailoringAuthRepo) FindCustomerByEmail(context.Context, string) (appauth.User, string, error) {
	return appauth.User{}, "", nil
}

func (r *tailoringAuthRepo) CreateSession(_ context.Context, _, _ string, tokenHash string, expiresAt time.Time) error {
	return nil
}

func (r *tailoringAuthRepo) FindSessionUser(_ context.Context, tokenHash string, now time.Time) (appauth.User, error) {
	if u, ok := r.users[tokenHash]; ok {
		return u, nil
	}
	return appauth.User{}, appauth.ErrSessionNotFound
}

func (r *tailoringAuthRepo) DeleteSession(context.Context, string) error { return nil }

func setupTailoringTestServer(service TailoringService, authRepo appauth.Repository, allowedOrigin ...string) http.Handler {
	var guard *AuthGuard
	origin := ""
	if len(allowedOrigin) > 0 {
		origin = allowedOrigin[0]
	}
	if authRepo != nil {
		guard = NewAuthGuard(appauth.NewService(authRepo, 24*time.Hour), AuthCookieConfig{
			Name:          "aodora_session",
			AllowedOrigin: origin,
		})
	} else {
		guard = NewAuthGuard(nil, AuthCookieConfig{
			Name:          "aodora_session",
			AllowedOrigin: origin,
		})
	}

	return NewRouter(nil, zerolog.Nop(), WithTailoring(service, guard, zerolog.Nop()))
}

func TestTailoringHandler_GuestMinimalSuccess(t *testing.T) {
	mockSvc := &mockTailoringService{}
	server := setupTailoringTestServer(mockSvc, nil)

	body := `{"full_name":"Nguyễn Văn A","phone":"0900000001"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tailoring-requests", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data map[string]string `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if resp.Data["id"] != "6d0000000000000000000001" {
		t.Errorf("expected id %q, got %q", "6d0000000000000000000001", resp.Data["id"])
	}
	if resp.Data["status"] != "submitted" {
		t.Errorf("expected status %q, got %q", "submitted", resp.Data["status"])
	}
	// Verify no contact data is exposed
	if _, ok := resp.Data["full_name"]; ok {
		t.Errorf("full_name must not be exposed in response")
	}
	if _, ok := resp.Data["phone"]; ok {
		t.Errorf("phone must not be exposed in response")
	}

	if mockSvc.lastInput.CustomerID != nil {
		t.Errorf("expected nil customer_id, got %v", *mockSvc.lastInput.CustomerID)
	}
	if mockSvc.lastInput.FullName != "Nguyễn Văn A" {
		t.Errorf("expected full_name %q, got %q", "Nguyễn Văn A", mockSvc.lastInput.FullName)
	}
}

func TestTailoringHandler_NameAliasSuccess(t *testing.T) {
	mockSvc := &mockTailoringService{}
	server := setupTailoringTestServer(mockSvc, nil)

	body := `{"name":"Trần Thị B","phone":"0900000002"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tailoring-requests", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", w.Code, w.Body.String())
	}
	if mockSvc.lastInput.FullName != "Trần Thị B" {
		t.Errorf("expected full_name %q, got %q", "Trần Thị B", mockSvc.lastInput.FullName)
	}
}

func TestTailoringHandler_AuthenticatedSuccess(t *testing.T) {
	mockSvc := &mockTailoringService{}
	authRepo := newTailoringAuthRepo()
	authRepo.addSession("valid-session-123", appauth.User{ID: "690000000000000000000001", Role: "customer"})
	server := setupTailoringTestServer(mockSvc, authRepo)

	body := `{"full_name":"Lê C","phone":"0900000003"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tailoring-requests", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "aodora_session", Value: "valid-session-123"})
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", w.Code, w.Body.String())
	}
	if mockSvc.lastInput.CustomerID == nil || *mockSvc.lastInput.CustomerID != "690000000000000000000001" {
		t.Errorf("expected customer_id 690000000000000000000001, got %v", mockSvc.lastInput.CustomerID)
	}
}

func TestTailoringHandler_InvalidCookie401(t *testing.T) {
	mockSvc := &mockTailoringService{}
	authRepo := newTailoringAuthRepo()
	server := setupTailoringTestServer(mockSvc, authRepo)

	body := `{"full_name":"Lê C","phone":"0900000003"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tailoring-requests", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "aodora_session", Value: "expired-or-bad-session"})
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d: %s", w.Code, w.Body.String())
	}
	var errResp map[string]apiError
	_ = json.Unmarshal(w.Body.Bytes(), &errResp)
	if errResp["error"].Code != "AUTH_REQUIRED" {
		t.Errorf("expected code AUTH_REQUIRED, got %q", errResp["error"].Code)
	}
}

func TestTailoringHandler_ValidationErrors(t *testing.T) {
	mockSvc := &mockTailoringService{
		errToRet: domaintailoring.NewInvalidFieldError("full_name", "Họ và tên là bắt buộc"),
	}
	server := setupTailoringTestServer(mockSvc, nil)

	tests := []struct {
		name       string
		body       string
		mockErr    error
		wantStatus int
		wantCode   string
		wantField  string
	}{
		{
			name:       "missing content-type",
			body:       `{"full_name":"A","phone":"0900000001"}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   "INVALID_BODY",
			wantField:  "Content-Type",
		},
		{
			name:       "invalid json",
			body:       `{"full_name":`,
			wantStatus: http.StatusBadRequest,
			wantCode:   "INVALID_BODY",
			wantField:  "body",
		},
		{
			name:       "unknown field disallowed",
			body:       `{"full_name":"A","phone":"0900000001","unknown_field":true}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   "INVALID_BODY",
			wantField:  "body",
		},
		{
			name:       "customer_id in body disallowed",
			body:       `{"full_name":"A","phone":"0900000001","customer_id":"690000000000000000000001"}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   "INVALID_BODY",
			wantField:  "body",
		},
		{
			name:       "status in body disallowed",
			body:       `{"full_name":"A","phone":"0900000001","status":"completed"}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   "INVALID_BODY",
			wantField:  "body",
		},
		{
			name:       "negative measurement",
			body:       `{"full_name":"A","phone":"0900000001","measurements":{"neck_cm":-5}}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   "INVALID_FIELD",
			wantField:  "neck_cm",
		},
		{
			name:       "string measurement",
			body:       `{"full_name":"A","phone":"0900000001","measurements":{"bust_cm":"abc"}}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   "INVALID_FIELD",
			wantField:  "bust_cm",
		},
		{
			name:       "zero measurement",
			body:       `{"full_name":"A","phone":"0900000001","measurements":{"waist_cm":0}}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   "INVALID_FIELD",
			wantField:  "waist_cm",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/tailoring-requests", bytes.NewBufferString(tt.body))
			if tt.name != "missing content-type" {
				req.Header.Set("Content-Type", "application/json")
			}
			w := httptest.NewRecorder()

			server.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d: %s", tt.wantStatus, w.Code, w.Body.String())
			}
			var errResp map[string]apiError
			_ = json.Unmarshal(w.Body.Bytes(), &errResp)
			if errResp["error"].Code != tt.wantCode {
				t.Errorf("expected code %q, got %q", tt.wantCode, errResp["error"].Code)
			}
			if tt.wantField != "" && errResp["error"].Field != tt.wantField {
				t.Errorf("expected field %q, got %q", tt.wantField, errResp["error"].Field)
			}
		})
	}
}

func TestTailoringHandler_CORSPreflight(t *testing.T) {
	mockSvc := &mockTailoringService{}
	server := setupTailoringTestServer(mockSvc, nil, "https://shop.example")

	// 1. Allowed origin preflight -> 204 No Content
	req := httptest.NewRequest(http.MethodOptions, "/api/v1/tailoring-requests", nil)
	req.Header.Set("Origin", "https://shop.example")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "Content-Type")
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204 for allowed origin, got %d", w.Code)
	}

	// 2. Unauthorized origin preflight -> 403 ORIGIN_FORBIDDEN
	reqEvil := httptest.NewRequest(http.MethodOptions, "/api/v1/tailoring-requests", nil)
	reqEvil.Header.Set("Origin", "https://evil.example")
	reqEvil.Header.Set("Access-Control-Request-Method", "POST")
	reqEvil.Header.Set("Access-Control-Request-Headers", "Content-Type")
	wEvil := httptest.NewRecorder()

	server.ServeHTTP(wEvil, reqEvil)

	if wEvil.Code != http.StatusForbidden {
		t.Errorf("expected 403 ORIGIN_FORBIDDEN for unauthorized origin, got %d", wEvil.Code)
	}
}

func TestTailoringHandler_InfraErrorIs503(t *testing.T) {
	mockSvc := &mockTailoringService{errToRet: domaintailoring.ErrTailoringUnavailable}
	server := setupTailoringTestServer(mockSvc, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tailoring-requests", strings.NewReader(`{"full_name":"A","phone":"0900000001"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d: %s", w.Code, w.Body.String())
	}
	var errResp map[string]apiError
	_ = json.Unmarshal(w.Body.Bytes(), &errResp)
	if errResp["error"].Code != "TAILORING_UNAVAILABLE" {
		t.Errorf("expected TAILORING_UNAVAILABLE, got %q", errResp["error"].Code)
	}
}

func TestTailoringHandler_ConflictingAliasesRejected(t *testing.T) {
	for _, body := range []string{
		`{"full_name":"A","name":"B","phone":"0900000001"}`,
		`{"full_name":"A","phone":"0900000001","notes":"x","note":"y"}`,
	} {
		mockSvc := &mockTailoringService{}
		server := setupTailoringTestServer(mockSvc, nil)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/tailoring-requests", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("%s: expected 400, got %d", body, w.Code)
		}
		if mockSvc.lastInput != nil {
			t.Fatalf("%s: service must not be called", body)
		}
	}
}
