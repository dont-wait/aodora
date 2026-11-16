package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/joho/godotenv"
	domainorder "github.com/your-org/aodora/server/internal/domain/order"
)

type HTTPConfig struct {
	Addr string
}

type MongoConfig struct {
	URI string
	DB  string
}

type LogConfig struct {
	Level  string
	Format string
}

type AuthConfig struct {
	SessionTTL    time.Duration
	CookieName    string
	CookieSecure  bool
	AllowedOrigin string
}

type OrderConfig struct {
	QuoteSecret    []byte
	QuoteTTL       time.Duration
	CODShippingFee *domainorder.Money
}

type Config struct {
	HTTP  HTTPConfig
	Log   LogConfig
	Mongo MongoConfig
	Auth  AuthConfig
	Order OrderConfig
}

type ConfigError struct {
	Name string
	Err  error
}

type ConfigErrors []ConfigError

func (e ConfigErrors) Error() string {
	if len(e) == 0 {
		return ""
	}
	parts := make([]string, 0, len(e))
	for _, item := range e {
		parts = append(parts, fmt.Sprintf("%s: %v", item.Name, item.Err))
	}
	return strings.Join(parts, "; ")
}

// LoadAllConfig loads every config group and keeps each error identifiable.
// Callers can use errors.As(err, &config.ConfigErrors) to log errors separately.
func LoadAllConfig() (Config, error) {
	var cfg Config
	var configErrors ConfigErrors

	if err := LoadEnv(); err != nil {
		configErrors = append(configErrors, ConfigError{Name: "environment", Err: err})
	}
	if value, err := LoadLogConfig(); err != nil {
		configErrors = append(configErrors, ConfigError{Name: "log", Err: err})
	} else {
		cfg.Log = value
	}
	if value, err := LoadHTTPConfig(); err != nil {
		configErrors = append(configErrors, ConfigError{Name: "http", Err: err})
	} else {
		cfg.HTTP = value
	}
	if value, err := LoadMongoConfig(); err != nil {
		configErrors = append(configErrors, ConfigError{Name: "mongo", Err: err})
	} else {
		cfg.Mongo = value
	}
	if value, err := LoadAuthConfig(); err != nil {
		configErrors = append(configErrors, ConfigError{Name: "auth", Err: err})
	} else {
		cfg.Auth = value
	}
	if value, err := LoadOrderConfig(); err != nil {
		configErrors = append(configErrors, ConfigError{Name: "order", Err: err})
	} else {
		cfg.Order = value
	}

	if len(configErrors) > 0 {
		return cfg, configErrors
	}
	return cfg, nil
}

func LoadOrderConfig() (OrderConfig, error) {
	var cfg OrderConfig

	secret := os.Getenv("ORDER_QUOTE_SECRET")
	if strings.TrimSpace(secret) != "" {
		secBytes := []byte(secret)
		if len(secBytes) < 32 {
			return OrderConfig{}, errors.New("ORDER_QUOTE_SECRET must be at least 32 bytes")
		}
		cfg.QuoteSecret = secBytes
	}

	ttlStr := envOr("ORDER_QUOTE_TTL", "15m")
	ttl, err := time.ParseDuration(ttlStr)
	if err != nil || ttl <= 0 {
		return OrderConfig{}, errors.New("ORDER_QUOTE_TTL must be a positive duration")
	}
	cfg.QuoteTTL = ttl

	feeStr := os.Getenv("ORDER_COD_SHIPPING_FEE")
	if strings.TrimSpace(feeStr) != "" {
		fee, err := domainorder.ParseMoney(strings.TrimSpace(feeStr))
		if err != nil {
			return OrderConfig{}, fmt.Errorf("ORDER_COD_SHIPPING_FEE is invalid: %w", err)
		}
		cfg.CODShippingFee = &fee
	}

	return cfg, nil
}

func LoadAuthConfig() (AuthConfig, error) {
	ttl, err := time.ParseDuration(envOr("AUTH_SESSION_TTL", "168h"))
	if err != nil || ttl <= 0 {
		return AuthConfig{}, errors.New("AUTH_SESSION_TTL must be a positive duration")
	}
	cookieName := strings.TrimSpace(envOr("AUTH_COOKIE_NAME", "aodora_session"))
	if cookieName == "" {
		return AuthConfig{}, errors.New("AUTH_COOKIE_NAME must not be empty")
	}
	secure := strings.EqualFold(envOr("AUTH_COOKIE_SECURE", "false"), "true")
	origin := strings.TrimSuffix(strings.TrimSpace(envOr("AUTH_ALLOWED_ORIGIN", "")), "/")
	return AuthConfig{SessionTTL: ttl, CookieName: cookieName, CookieSecure: secure, AllowedOrigin: origin}, nil
}

// LoadEnv imports .env when present. Real environment variables keep priority.
func LoadEnv() error {
	return loadDotEnv()
}

func LoadHTTPConfig() (HTTPConfig, error) {
	addr := envOr("HTTP_ADDR", ":8080")
	if strings.TrimSpace(addr) == "" {
		return HTTPConfig{}, errors.New("HTTP_ADDR must not be empty")
	}
	return HTTPConfig{Addr: addr}, nil
}

func LoadLogConfig() (LogConfig, error) {
	level := strings.ToLower(envOr("LOG_LEVEL", "info"))
	if !map[string]bool{"trace": true, "debug": true, "info": true, "warn": true, "error": true, "fatal": true, "panic": true}[level] {
		return LogConfig{}, fmt.Errorf("unsupported LOG_LEVEL %q", level)
	}
	format := strings.ToLower(envOr("LOG_FORMAT", "json"))
	if format != "json" && format != "text" {
		return LogConfig{}, fmt.Errorf("unsupported LOG_FORMAT %q", format)
	}
	return LogConfig{Level: level, Format: format}, nil
}

func LoadMongoConfig() (MongoConfig, error) {
	cfg := MongoConfig{
		URI: envOr("MONGO_URI", "mongodb://localhost:27017"),
		DB:  envOr("MONGO_DB", "QuanLyBanQuanAo"),
	}
	if strings.TrimSpace(cfg.URI) == "" {
		return MongoConfig{}, errors.New("MONGO_URI must not be empty")
	}
	if strings.TrimSpace(cfg.DB) == "" {
		return MongoConfig{}, errors.New("MONGO_DB must not be empty")
	}
	return cfg, nil
}

func loadDotEnv() error {
	root, err := findProjectRoot()
	if err != nil {
		return fmt.Errorf("find project root: %w", err)
	}
	path := filepath.Join(root, ".env")
	if err := godotenv.Load(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("load env file %q: %w", path, err)
	}
	return nil
}

func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("go.mod not found")
		}
		dir = parent
	}
}

func envOr(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}
