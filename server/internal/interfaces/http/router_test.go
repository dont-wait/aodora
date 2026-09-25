package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/your-org/aodora/server/internal/application/health"
	"github.com/your-org/aodora/server/internal/logger"
)

type healthyDatabase struct{}

func (healthyDatabase) Ping(context.Context) error { return nil }

func TestHealthz(t *testing.T) {
	req := httptest.NewRequest("GET", "/healthz", nil)
	rec := httptest.NewRecorder()
	NewRouter(nil, *logger.NewDiscard()).ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestReadyz(t *testing.T) {
	service := health.NewService(healthyDatabase{}, *logger.NewDiscard())
	req := httptest.NewRequest("GET", "/readyz", nil)
	rec := httptest.NewRecorder()
	NewRouter(service, *logger.NewDiscard()).ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestStatusWriterDefaultsToOK(t *testing.T) {
	recorder := httptest.NewRecorder()
	writer := &statusWriter{ResponseWriter: recorder, status: http.StatusOK}
	_, _ = writer.Write([]byte("ok"))
	if writer.status != http.StatusOK {
		t.Fatalf("status = %d, want %d", writer.status, http.StatusOK)
	}
}
