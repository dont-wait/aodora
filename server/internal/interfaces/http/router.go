package httpapi

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/rs/zerolog"
	"github.com/your-org/aodora/server/internal/application/health"
)

// RouterOption registers an isolated capability without changing other routes.
type RouterOption func(*http.ServeMux)

func NewRouter(healthService *health.Service, log zerolog.Logger, extra ...RouterOption) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthz)
	mux.HandleFunc("GET /readyz", readyz(healthService))
	for _, register := range extra {
		if register != nil {
			register(mux)
		}
	}
	return requestLogger(log, withJSONContentType(mux))
}

func requestLogger(log zerolog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		writer := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(writer, r)
		log.Info().
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Int("status", writer.status).
			Int64("duration_ms", time.Since(started).Milliseconds()).
			Msg("http request")
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func readyz(service *health.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := service.Check(r.Context()); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "unavailable"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	}
}

func withJSONContentType(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	// Encode before WriteHeader: failure must not produce HTTP 200 with
	// an incomplete or invalid JSON document.
	data, err := json.Marshal(value)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":{"code":"RESPONSE_ENCODING_FAILED","message":"Không thể tạo phản hồi"}}`))
		return
	}
	w.WriteHeader(status)
	_, _ = w.Write(append(data, '\n'))
}
