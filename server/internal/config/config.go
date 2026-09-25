package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
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

type Config struct {
	HTTP  HTTPConfig
	Log   LogConfig
	Mongo MongoConfig
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

	if len(configErrors) > 0 {
		return cfg, configErrors
	}
	return cfg, nil
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
		DB:  envOr("MONGO_DB", "aodora"),
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
