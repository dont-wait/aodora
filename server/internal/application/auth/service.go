package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailTaken         = errors.New("email already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrSessionNotFound    = errors.New("session not found")
	ErrUserNotFound       = errors.New("user not found")
)

type Invalid struct {
	Field   string
	Message string
}

func (e Invalid) Error() string { return e.Field + ": " + e.Message }

type User struct {
	ID       string `json:"id"`
	FullName string `json:"full_name"`
	Email    string `json:"email"`
	Phone    string `json:"phone,omitempty"`
	Role     string `json:"role"`
}

type RegisterInput struct {
	FullName string `json:"full_name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Phone    string `json:"phone"`
}

type Repository interface {
	CreateCustomer(context.Context, RegisterInput, string) (User, error)
	FindCustomerByEmail(context.Context, string) (User, string, error)
	CreateSession(ctx context.Context, userID, role, tokenHash string, expiresAt time.Time) error
	FindSessionUser(context.Context, string, time.Time) (User, error)
	DeleteSession(context.Context, string) error
}

type Service struct {
	repo       Repository
	sessionTTL time.Duration
	now        func() time.Time
}

func NewService(repo Repository, sessionTTL time.Duration) *Service {
	if sessionTTL <= 0 {
		sessionTTL = 7 * 24 * time.Hour
	}
	return &Service{repo: repo, sessionTTL: sessionTTL, now: time.Now}
}

func NormalizeEmail(value string) string { return strings.ToLower(strings.TrimSpace(value)) }

func ValidateEmail(value string) error { return validateEmail(NormalizeEmail(value)) }

func (s *Service) Register(ctx context.Context, input RegisterInput) (User, error) {
	input.FullName = strings.TrimSpace(input.FullName)
	input.Email = NormalizeEmail(input.Email)
	input.Phone = strings.TrimSpace(input.Phone)
	if utf8.RuneCountInString(input.FullName) < 1 || utf8.RuneCountInString(input.FullName) > 150 {
		return User{}, Invalid{Field: "full_name", Message: "Họ tên phải từ 1 đến 150 ký tự"}
	}
	if err := validateEmail(input.Email); err != nil {
		return User{}, err
	}
	if len(input.Password) < 8 || len(input.Password) > 72 {
		return User{}, Invalid{Field: "password", Message: "Mật khẩu phải từ 8 đến 72 ký tự"}
	}
	if utf8.RuneCountInString(input.Phone) > 30 {
		return User{}, Invalid{Field: "phone", Message: "Số điện thoại quá dài"}
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return User{}, err
	}
	return s.repo.CreateCustomer(ctx, input, string(hash))
}

func (s *Service) Login(ctx context.Context, email, password string) (User, string, time.Time, error) {
	email = NormalizeEmail(email)
	if err := validateEmail(email); err != nil || password == "" {
		return User{}, "", time.Time{}, ErrInvalidCredentials
	}
	user, passwordHash, err := s.repo.FindCustomerByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return User{}, "", time.Time{}, ErrInvalidCredentials
		}
		return User{}, "", time.Time{}, err
	}
	if bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)) != nil {
		return User{}, "", time.Time{}, ErrInvalidCredentials
	}
	token, err := newToken()
	if err != nil {
		return User{}, "", time.Time{}, err
	}
	expiresAt := s.now().Add(s.sessionTTL)
	if err := s.repo.CreateSession(ctx, user.ID, user.Role, hashToken(token), expiresAt); err != nil {
		return User{}, "", time.Time{}, err
	}
	return user, token, expiresAt, nil
}

func (s *Service) Authenticate(ctx context.Context, token string) (User, error) {
	if token == "" {
		return User{}, ErrSessionNotFound
	}
	return s.repo.FindSessionUser(ctx, hashToken(token), s.now())
}

func (s *Service) Logout(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	return s.repo.DeleteSession(ctx, hashToken(token))
}

func SessionCookieMaxAge(ttl time.Duration) int { return int(ttl.Seconds()) }

func validateEmail(value string) error {
	if len(value) > 254 {
		return Invalid{Field: "email", Message: "Email quá dài"}
	}
	address, err := mail.ParseAddress(value)
	if err != nil || address.Address != value || !strings.Contains(value, "@") {
		return Invalid{Field: "email", Message: "Email không hợp lệ"}
	}
	return nil
}

func newToken() (string, error) {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return hex.EncodeToString(value), nil
}

func hashToken(token string) string {
	digest := sha256.Sum256([]byte(token))
	return hex.EncodeToString(digest[:])
}
