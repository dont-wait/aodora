package config

import (
	"strings"
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

func TestLoadOrderConfig(t *testing.T) {
	// 1. Missing secret and fee -> defaults/nil, no error
	t.Setenv("ORDER_QUOTE_SECRET", "")
	t.Setenv("ORDER_QUOTE_TTL", "")
	t.Setenv("ORDER_COD_SHIPPING_FEE", "")
	cfg, err := LoadOrderConfig()
	if err != nil {
		t.Fatalf("unexpected error for empty config: %v", err)
	}
	if cfg.QuoteSecret != nil {
		t.Errorf("expected QuoteSecret to be nil, got %v", cfg.QuoteSecret)
	}
	if cfg.QuoteTTL != 15*time.Minute {
		t.Errorf("expected QuoteTTL to default to 15m, got %v", cfg.QuoteTTL)
	}
	if cfg.CODShippingFee != nil {
		t.Errorf("expected CODShippingFee to be nil, got %v", cfg.CODShippingFee)
	}

	// 2. Secret < 32 bytes -> error and message does NOT leak secret
	shortSecret := "super-secret-short"
	t.Setenv("ORDER_QUOTE_SECRET", shortSecret)
	_, err = LoadOrderConfig()
	if err == nil {
		t.Fatal("expected error for secret < 32 bytes, got nil")
	}
	if strings.Contains(err.Error(), shortSecret) {
		t.Errorf("error message leaked secret value: %s", err.Error())
	}

	// 3. Secret >= 32 bytes -> valid
	validSecret := "12345678901234567890123456789012"
	t.Setenv("ORDER_QUOTE_SECRET", validSecret)
	cfg, err = LoadOrderConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(cfg.QuoteSecret) != validSecret {
		t.Errorf("QuoteSecret = %s, want %s", string(cfg.QuoteSecret), validSecret)
	}

	// 4. Invalid TTL
	t.Setenv("ORDER_QUOTE_TTL", "invalid")
	if _, err := LoadOrderConfig(); err == nil {
		t.Error("expected error for invalid TTL, got nil")
	}
	t.Setenv("ORDER_QUOTE_TTL", "-5m")
	if _, err := LoadOrderConfig(); err == nil {
		t.Error("expected error for negative TTL, got nil")
	}
	t.Setenv("ORDER_QUOTE_TTL", "0s")
	if _, err := LoadOrderConfig(); err == nil {
		t.Error("expected error for 0s TTL, got nil")
	}

	// Valid custom TTL
	t.Setenv("ORDER_QUOTE_TTL", "30m")
	cfg, err = LoadOrderConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.QuoteTTL != 30*time.Minute {
		t.Errorf("QuoteTTL = %v, want 30m", cfg.QuoteTTL)
	}

	// 5. Invalid COD fee
	t.Setenv("ORDER_COD_SHIPPING_FEE", "-100")
	if _, err := LoadOrderConfig(); err == nil {
		t.Error("expected error for negative COD fee, got nil")
	}
	t.Setenv("ORDER_COD_SHIPPING_FEE", "invalid")
	if _, err := LoadOrderConfig(); err == nil {
		t.Error("expected error for invalid COD fee, got nil")
	}

	// 6. COD fee = 0 is valid and non-nil
	t.Setenv("ORDER_COD_SHIPPING_FEE", "0")
	cfg, err = LoadOrderConfig()
	if err != nil {
		t.Fatalf("unexpected error for 0 fee: %v", err)
	}
	if cfg.CODShippingFee == nil || cfg.CODShippingFee.String() != "0" {
		t.Errorf("CODShippingFee = %v, want 0", cfg.CODShippingFee)
	}

	// 7. COD fee = 30000
	t.Setenv("ORDER_COD_SHIPPING_FEE", "30000")
	cfg, err = LoadOrderConfig()
	if err != nil {
		t.Fatalf("unexpected error for 30000 fee: %v", err)
	}
	if cfg.CODShippingFee == nil || cfg.CODShippingFee.String() != "30000" {
		t.Errorf("CODShippingFee = %v, want 30000", cfg.CODShippingFee)
	}
}
