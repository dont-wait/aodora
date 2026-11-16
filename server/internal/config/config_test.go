package config

import (
	"testing"
	"time"
)

func TestLoadAuthConfig(t *testing.T) {
	t.Setenv("AUTH_SESSION_TTL", "2h")
	t.Setenv("AUTH_COOKIE_NAME", "test_session")
	t.Setenv("AUTH_COOKIE_SECURE", "true")
	cfg, err := LoadAuthConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.SessionTTL != 2*time.Hour || cfg.CookieName != "test_session" || !cfg.CookieSecure {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}
