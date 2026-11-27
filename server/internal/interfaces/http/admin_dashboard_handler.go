package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/rs/zerolog"
	app "github.com/your-org/aodora/server/internal/application/dashboard"
)

// AdminDashboardService is the read model behind GET /api/v1/admin/dashboard.
type AdminDashboardService interface {
	Report(ctx context.Context, from, to, granularity string, lowStock int64) (app.Report, error)
}

type adminDashboardHandler struct {
	service AdminDashboardService
	log     zerolog.Logger
}

func WithAdminDashboard(service AdminDashboardService, authorize AdminAuthorizer, log zerolog.Logger, allowedOrigin string) RouterOption {
	return func(mux *http.ServeMux) {
		h := adminDashboardHandler{service: service, log: log}
		// Shares the catalog guard; its OPTIONS /api/v1/admin/{path...} serves preflight.
		guard := adminCatalogHandler{authorize: authorize, log: log, allowedOrigin: strings.TrimSuffix(strings.TrimSpace(allowedOrigin), "/")}
		mux.HandleFunc("GET /api/v1/admin/dashboard", guard.guard(h.get))
	}
}

func (h adminDashboardHandler) get(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	for key, values := range query {
		switch key {
		case "from", "to", "granularity", "low_stock":
			if len(values) != 1 {
				respondError(w, http.StatusBadRequest, "INVALID_QUERY", key, "Tham số bị lặp")
				return
			}
		default:
			respondError(w, http.StatusBadRequest, "INVALID_QUERY", key, "Tham số không hợp lệ")
			return
		}
	}
	var lowStock int64
	if raw := query.Get("low_stock"); raw != "" {
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			respondError(w, http.StatusBadRequest, "INVALID_QUERY", "low_stock", "Phải là số nguyên")
			return
		}
		lowStock = n
	}
	report, err := h.service.Report(r.Context(), query.Get("from"), query.Get("to"), query.Get("granularity"), lowStock)
	if err != nil {
		var invalid app.Invalid
		switch {
		case errors.As(err, &invalid):
			respondError(w, http.StatusBadRequest, "INVALID_QUERY", invalid.Field, invalid.Message)
		case errors.Is(err, context.DeadlineExceeded):
			respondError(w, http.StatusGatewayTimeout, "DASHBOARD_TIMEOUT", "", "Truy vấn thống kê quá thời gian")
		default:
			h.log.Error().Str("cause", safeCatalogError(err)).Msg("dashboard failed")
			respondError(w, http.StatusInternalServerError, "DASHBOARD_UNAVAILABLE", "", "Không thể tải số liệu thống kê")
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": report})
}
