package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	app "github.com/your-org/aodora/server/internal/application/auth"
)

type authContextKey struct{}

type AuthCookieConfig struct {
	Name          string
	Secure        bool
	TTL           time.Duration
	AllowedOrigin string
}

type AuthGuard struct {
	service *app.Service
	cookie  AuthCookieConfig
}

func NewAuthGuard(service *app.Service, cookie AuthCookieConfig) *AuthGuard {
	if cookie.Name == "" {
		cookie.Name = "aodora_session"
	}
	if cookie.TTL <= 0 {
		cookie.TTL = 7 * 24 * time.Hour
	}
	return &AuthGuard{service: service, cookie: cookie}
}

func WithAuth(guard *AuthGuard) RouterOption {
	return func(mux *http.ServeMux) {
		h := authHandler{guard: guard}
		mux.HandleFunc("OPTIONS /api/v1/auth/{path...}", h.preflight)
		mux.HandleFunc("POST /api/v1/auth/register", h.withOrigin(h.register))
		mux.HandleFunc("POST /api/v1/auth/login", h.withOrigin(h.login))
		mux.HandleFunc("POST /api/v1/auth/logout", h.withOrigin(h.logout))
		mux.HandleFunc("GET /api/v1/auth/me", h.withOrigin(h.me))
	}
}

func (g *AuthGuard) RequireAdmin(r *http.Request) bool {
	user, ok := g.authenticate(r)
	return ok && user.Role == "admin"
}

func (g *AuthGuard) RequireAuth(r *http.Request) bool {
	_, ok := g.authenticate(r)
	return ok
}

func (g *AuthGuard) Identity(r *http.Request) (app.User, bool) {
	user, ok := r.Context().Value(authContextKey{}).(app.User)
	if ok {
		return user, true
	}
	return g.authenticate(r)
}

func (g *AuthGuard) authenticate(r *http.Request) (app.User, bool) {
	if user, ok := r.Context().Value(authContextKey{}).(app.User); ok {
		return user, true
	}
	cookie, err := r.Cookie(g.cookie.Name)
	if err != nil || cookie.Value == "" {
		return app.User{}, false
	}
	user, err := g.service.Authenticate(r.Context(), cookie.Value)
	if err != nil {
		return app.User{}, false
	}
	*r = *r.WithContext(context.WithValue(r.Context(), authContextKey{}, user))
	return user, true
}

type authHandler struct{ guard *AuthGuard }

func (h authHandler) withOrigin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !h.cors(w, r) {
			respondError(w, http.StatusForbidden, "ORIGIN_FORBIDDEN", "", "Nguồn truy cập không được cấp phép")
			return
		}
		next(w, r)
	}
}

func (h authHandler) cors(w http.ResponseWriter, r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	parsed, err := url.Parse(origin)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.Path != "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return false
	}
	if h.guard.cookie.AllowedOrigin == origin {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Add("Vary", "Origin")
		return true
	}
	return parsed.Host == r.Host && (parsed.Scheme == "http" || r.TLS != nil)
}

func (h authHandler) preflight(w http.ResponseWriter, r *http.Request) {
	if !h.cors(w, r) || r.Header.Get("Origin") == "" {
		respondError(w, http.StatusForbidden, "ORIGIN_FORBIDDEN", "", "Nguồn truy cập không được cấp phép")
		return
	}
	if !slices.Contains([]string{"GET", "POST"}, r.Header.Get("Access-Control-Request-Method")) {
		respondError(w, http.StatusBadRequest, "INVALID_PREFLIGHT", "", "Phương thức không được hỗ trợ")
		return
	}
	for _, header := range strings.Split(r.Header.Get("Access-Control-Request-Headers"), ",") {
		header = strings.TrimSpace(strings.ToLower(header))
		if header != "" && header != "content-type" && header != "accept" {
			respondError(w, http.StatusBadRequest, "INVALID_PREFLIGHT", "", "Header không được hỗ trợ")
			return
		}
	}
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept")
	w.Header().Set("Access-Control-Max-Age", "600")
	w.WriteHeader(http.StatusNoContent)
}

type registerRequest struct {
	FullName string `json:"full_name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Phone    string `json:"phone"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func decodeAuthJSON(w http.ResponseWriter, r *http.Request, out any) bool {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "Content-Type", "Yêu cầu application/json")
		return false
	}
	decoder := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "body", "JSON không hợp lệ")
		return false
	}
	return true
}

func (h authHandler) register(w http.ResponseWriter, r *http.Request) {
	var input registerRequest
	if !decodeAuthJSON(w, r, &input) {
		return
	}
	user, err := h.guard.service.Register(r.Context(), app.RegisterInput(input))
	if err != nil {
		h.respondError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"data": user})
}

func (h authHandler) login(w http.ResponseWriter, r *http.Request) {
	var input loginRequest
	if !decodeAuthJSON(w, r, &input) {
		return
	}
	user, token, expiresAt, err := h.guard.service.Login(r.Context(), input.Email, input.Password)
	if err != nil {
		h.respondError(w, err)
		return
	}
	h.setCookie(w, token, expiresAt)
	writeJSON(w, http.StatusOK, map[string]any{"data": user})
}

func (h authHandler) logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(h.guard.cookie.Name); err == nil {
		if err := h.guard.service.Logout(r.Context(), cookie.Value); err != nil {
			respondError(w, http.StatusInternalServerError, "AUTH_UNAVAILABLE", "", "Không thể đăng xuất")
			return
		}
	}
	h.clearCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (h authHandler) me(w http.ResponseWriter, r *http.Request) {
	user, ok := h.guard.Identity(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "AUTH_REQUIRED", "", "Vui lòng đăng nhập")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": user})
}

func (h authHandler) setCookie(w http.ResponseWriter, token string, expiresAt time.Time) {
	sameSite := http.SameSiteLaxMode
	if h.guard.cookie.AllowedOrigin != "" && h.guard.cookie.Secure {
		sameSite = http.SameSiteNoneMode
	}
	http.SetCookie(w, &http.Cookie{
		Name: h.guard.cookie.Name, Value: token, Path: "/", HttpOnly: true,
		Secure: h.guard.cookie.Secure, SameSite: sameSite,
		Expires: expiresAt, MaxAge: app.SessionCookieMaxAge(h.guard.cookie.TTL),
	})
}

func (h authHandler) clearCookie(w http.ResponseWriter) {
	sameSite := http.SameSiteLaxMode
	if h.guard.cookie.AllowedOrigin != "" && h.guard.cookie.Secure {
		sameSite = http.SameSiteNoneMode
	}
	http.SetCookie(w, &http.Cookie{Name: h.guard.cookie.Name, Value: "", Path: "/", HttpOnly: true, Secure: h.guard.cookie.Secure, SameSite: sameSite, MaxAge: -1, Expires: time.Unix(1, 0)})
}

func (h authHandler) respondError(w http.ResponseWriter, err error) {
	var invalid app.Invalid
	switch {
	case errors.As(err, &invalid):
		respondError(w, http.StatusBadRequest, "INVALID_FIELD", invalid.Field, invalid.Message)
	case errors.Is(err, app.ErrEmailTaken):
		respondError(w, http.StatusConflict, "EMAIL_TAKEN", "email", "Email đã được sử dụng")
	case errors.Is(err, app.ErrInvalidCredentials):
		respondError(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "", "Email hoặc mật khẩu không đúng")
	default:
		respondError(w, http.StatusInternalServerError, "AUTH_UNAVAILABLE", "", "Không thể xử lý xác thực")
	}
}
