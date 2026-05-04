package app

import (
	"log/slog"
	"net/http"

	"charging-ops/backend/internal/common/health"
	"charging-ops/backend/internal/common/response"
	"charging-ops/backend/internal/domain/session"
	"charging-ops/backend/internal/domain/site"
)

func registerRoutes(
	mux *http.ServeMux,
	healthHandler *health.Handler,
	siteHandler *site.Handler,
	sessionHandler *session.Handler,
) {
	mux.HandleFunc("GET /api/v1/health", healthHandler.Check)
	mux.HandleFunc("GET /api/v1/meta", handleMeta)
	mux.HandleFunc("GET /api/v1/sites", siteHandler.List)
	mux.HandleFunc("GET /api/v1/sites/{siteId}/topology", siteHandler.Topology)
	mux.HandleFunc("GET /api/v1/sessions", sessionHandler.List)
	mux.HandleFunc("GET /api/v1/sessions/{sessionId}", sessionHandler.Detail)
	mux.HandleFunc("POST /api/v1/sessions/{sessionId}/transition", sessionHandler.Transition)
}

func handleMeta(w http.ResponseWriter, r *http.Request) {
	response.OK(w, r, map[string]any{
		"name":    "charging-station-virtual-ops",
		"version": "0.1.0",
		"modules": []string{
			"protocol-gateway",
			"site-charger",
			"session",
			"load-control",
			"billing-reconciliation",
			"maintenance-sla",
			"audit-permission",
			"ai-assistant",
			"operations-dashboard",
		},
	})
}

func traceMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		traceID := r.Header.Get("X-Trace-Id")
		if traceID == "" {
			traceID = response.NewTraceID()
		}

		ctx := response.WithTraceID(r.Context(), traceID)
		logger.Info("http request", "module", "http", "traceId", traceID, "method", r.Method, "path", r.URL.Path)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
