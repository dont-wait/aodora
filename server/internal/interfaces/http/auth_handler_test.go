package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	app "github.com/your-org/aodora/server/internal/application/auth"
	"github.com/your-org/aodora/server/internal/logger"
)

type authHTTPRepository struct {
	user      app.User
	password  string
	tokenHash string
	expiresAt time.Time
}

func (r *authHTTPRepository) CreateCustomer(_ context.Context, input app.RegisterInput, passwordHash string) (app.User, error) {
	r.password = passwordHash
	r.user = app.User{ID: "670000000000000000000001", FullName: input.FullName, Email: input.Email, Role: "customer"}
	return r.user, nil
}
func (r *authHTTPRepository) FindCustomerByEmail(context.Context, string) (app.User, string, error) {
	if r.user.ID == "" {
		return app.User{}, "", app.ErrUserNotFound
	}
	return r.user, r.password, nil
}
func (r *authHTTPRepository) CreateSession(_ context.Context, _, _ string, tokenHash string, expiresAt time.Time) error {
	r.tokenHash, r.expiresAt = tokenHash, expiresAt
	return nil
}
func (r *authHTTPRepository) FindSessionUser(_ context.Context, tokenHash string, now time.Time) (app.User, error) {
	if tokenHash != r.tokenHash || !r.expiresAt.After(now) {
		return app.User{}, app.ErrSessionNotFound
	}
	return r.user, nil
}
func (r *authHTTPRepository) DeleteSession(context.Context, string) error { return nil }

func TestAuthRoutesRegisterLoginMe(t *testing.T) {
	repo := &authHTTPRepository{}
	service := app.NewService(repo, time.Hour)
	guard := NewAuthGuard(service, AuthCookieConfig{Name: "session", TTL: time.Hour})
	router := NewRouter(nil, *logger.NewDiscard(), WithAuth(guard))

	record := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(`{"full_name":"Owner","email":"owner@example.com","password":"password-123"}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(record, request)
	if record.Code != http.StatusCreated {
		t.Fatalf("register status = %d, body = %s", record.Code, record.Body.String())
	}

	record = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"email":"owner@example.com","password":"password-123"}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(record, request)
	if record.Code != http.StatusOK {
		t.Fatalf("login status = %d, body = %s", record.Code, record.Body.String())
	}
	cookies := record.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != "session" || !cookies[0].HttpOnly {
		t.Fatalf("unexpected session cookie: %+v", cookies)
	}

	record = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	request.AddCookie(cookies[0])
	router.ServeHTTP(record, request)
	if record.Code != http.StatusOK || !strings.Contains(record.Body.String(), "owner@example.com") {
		t.Fatalf("me status = %d, body = %s", record.Code, record.Body.String())
	}
}

func TestAuthMeRequiresSession(t *testing.T) {
	guard := NewAuthGuard(app.NewService(&authHTTPRepository{}, time.Hour), AuthCookieConfig{Name: "session", TTL: time.Hour})
	router := NewRouter(nil, *logger.NewDiscard(), WithAuth(guard))
	record := httptest.NewRecorder()
	router.ServeHTTP(record, httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil))
	if record.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, body = %s", record.Code, record.Body.String())
	}
}

func TestAuthPreflightAllowsConfiguredOriginOnly(t *testing.T) {
	guard := NewAuthGuard(app.NewService(&authHTTPRepository{}, time.Hour), AuthCookieConfig{Name: "session", TTL: time.Hour, AllowedOrigin: "https://shop.example"})
	router := NewRouter(nil, *logger.NewDiscard(), WithAuth(guard))

	record := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodOptions, "/api/v1/auth/login", nil)
	request.Header.Set("Origin", "https://shop.example")
	request.Header.Set("Access-Control-Request-Method", http.MethodPost)
	request.Header.Set("Access-Control-Request-Headers", "content-type")
	router.ServeHTTP(record, request)
	if record.Code != http.StatusNoContent || record.Header().Get("Access-Control-Allow-Origin") != "https://shop.example" {
		t.Fatalf("allowed preflight = %d, headers = %v", record.Code, record.Header())
	}

	record = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodOptions, "/api/v1/auth/login", nil)
	request.Header.Set("Origin", "https://evil.example")
	request.Header.Set("Access-Control-Request-Method", http.MethodPost)
	router.ServeHTTP(record, request)
	if record.Code != http.StatusForbidden {
		t.Fatalf("forbidden preflight status = %d", record.Code)
	}
}

func TestAdminAuthorizationUsesAuthenticatedRole(t *testing.T) {
	repo := &authHTTPRepository{}
	service := app.NewService(repo, time.Hour)
	if _, err := service.Register(context.Background(), app.RegisterInput{FullName: "Owner", Email: "owner@example.com", Password: "password-123"}); err != nil {
		t.Fatal(err)
	}
	guard := NewAuthGuard(service, AuthCookieConfig{Name: "session", TTL: time.Hour})
	_, token, _, err := service.Login(context.Background(), "owner@example.com", "password-123")
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/admin/categories", nil)
	request.AddCookie(&http.Cookie{Name: "session", Value: token})
	if guard.RequireAdmin(request) {
		t.Fatal("customer was authorized as admin")
	}
	repo.user.Role = "admin"
	request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/categories", nil)
	request.AddCookie(&http.Cookie{Name: "session", Value: token})
	if !guard.RequireAdmin(request) {
		t.Fatal("admin was not authorized")
	}
}
