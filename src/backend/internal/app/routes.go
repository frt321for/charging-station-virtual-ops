package app

import (
	"log/slog"
	"net/http"

	"charging-ops/backend/internal/common/health"
	"charging-ops/backend/internal/common/response"
	"charging-ops/backend/internal/domain/command"
	"charging-ops/backend/internal/domain/gateway"
	"charging-ops/backend/internal/domain/session"
	"charging-ops/backend/internal/domain/site"
)

func registerRoutes(
	mux *http.ServeMux,
	healthHandler *health.Handler,
	siteHandler *site.Handler,
	sessionHandler *session.Handler,
	commandHandler *command.Handler,
	gatewayHandler *gateway.Handler,
) {
	mux.HandleFunc("GET /api/v1/health", healthHandler.Check)
	mux.HandleFunc("GET /api/v1/meta", handleMeta)
	mux.HandleFunc("GET /api/v1/sites", siteHandler.List)
	mux.HandleFunc("GET /api/v1/sites/{siteId}/topology", siteHandler.Topology)
	mux.HandleFunc("POST /api/v1/reservations", sessionHandler.CreateReservation)
	mux.HandleFunc("GET /api/v1/sessions", sessionHandler.List)
	mux.HandleFunc("GET /api/v1/sessions/{sessionId}", sessionHandler.Detail)
	mux.HandleFunc("POST /api/v1/sessions/{sessionId}/transition", sessionHandler.Transition)
	mux.HandleFunc("POST /api/v1/sessions/{sessionId}/commands/start", commandHandler.Create(command.TypeStart))
	mux.HandleFunc("POST /api/v1/sessions/{sessionId}/commands/stop", commandHandler.Create(command.TypeStop))
	mux.HandleFunc("POST /api/v1/sessions/{sessionId}/commands/pause", commandHandler.Create(command.TypePause))
	mux.HandleFunc("POST /api/v1/sessions/{sessionId}/commands/resume", commandHandler.Create(command.TypeResume))
	mux.HandleFunc("POST /api/v1/sessions/{sessionId}/commands/reset", commandHandler.Create(command.TypeReset))
	mux.HandleFunc("POST /api/v1/sessions/{sessionId}/commands/limit-power", commandHandler.Create(command.TypeLimitPower))
	mux.HandleFunc("POST /api/v1/gateway/chargers/register", gatewayHandler.Register)
	mux.HandleFunc("POST /api/v1/gateway/chargers/{chargerCode}/heartbeat", gatewayHandler.Heartbeat)
	mux.HandleFunc("POST /api/v1/gateway/chargers/{chargerCode}/status", gatewayHandler.Status)
	mux.HandleFunc("POST /api/v1/gateway/chargers/{chargerCode}/meter-values", gatewayHandler.MeterValue)
	mux.HandleFunc("POST /api/v1/gateway/chargers/{chargerCode}/alarms", gatewayHandler.Alarm)
	mux.HandleFunc("POST /api/v1/gateway/chargers/{chargerCode}/command-receipts", commandHandler.Receipt)
	mux.HandleFunc("POST /api/v1/gateway/chargers/{chargerCode}/offline", gatewayHandler.Offline)
	mux.HandleFunc("/", handleNotFound)
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

func handleNotFound(w http.ResponseWriter, r *http.Request) {
	response.Error(w, r, http.StatusNotFound, 10004, "接口不存在", "route not found")
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
