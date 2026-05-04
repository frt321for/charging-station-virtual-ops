package app

import (
	"log/slog"
	"net/http"

	"charging-ops/backend/internal/common/health"
	"charging-ops/backend/internal/common/response"
	"charging-ops/backend/internal/domain/aiops"
	"charging-ops/backend/internal/domain/audit"
	"charging-ops/backend/internal/domain/auth"
	"charging-ops/backend/internal/domain/billing"
	"charging-ops/backend/internal/domain/command"
	"charging-ops/backend/internal/domain/configmgmt"
	"charging-ops/backend/internal/domain/finance"
	"charging-ops/backend/internal/domain/gateway"
	"charging-ops/backend/internal/domain/loadcontrol"
	"charging-ops/backend/internal/domain/maintenance"
	"charging-ops/backend/internal/domain/session"
	"charging-ops/backend/internal/domain/simcontrol"
	"charging-ops/backend/internal/domain/site"
)

func registerRoutes(
	mux *http.ServeMux,
	healthHandler *health.Handler,
	siteHandler *site.Handler,
	sessionHandler *session.Handler,
	commandHandler *command.Handler,
	gatewayHandler *gateway.Handler,
	billingHandler *billing.Handler,
	loadControlHandler *loadcontrol.Handler,
	maintenanceHandler *maintenance.Handler,
	financeHandler *finance.Handler,
	simulatorHandler *simcontrol.Handler,
	authHandler *auth.Handler,
	authService *auth.Service,
	auditHandler *audit.Handler,
	aiHandler *aiops.Handler,
	configHandler *configmgmt.Handler,
) {
	mux.HandleFunc("GET /api/v1/health", healthHandler.Check)
	mux.HandleFunc("GET /api/v1/meta", handleMeta)
	mux.HandleFunc("POST /api/v1/auth/login", authHandler.Login)
	mux.HandleFunc("GET /api/v1/auth/me", authService.Require("", authHandler.Me))
	mux.HandleFunc("POST /api/v1/auth/logout", authService.Require("", authHandler.Logout))
	mux.HandleFunc("GET /api/v1/users", authService.Require(auth.PermissionUsersManage, authHandler.ListUsers))
	mux.HandleFunc("GET /api/v1/roles", authService.Require(auth.PermissionUsersManage, authHandler.ListRoles))
	mux.HandleFunc("GET /api/v1/sites", authService.Require(auth.PermissionSitesRead, siteHandler.List))
	mux.HandleFunc("GET /api/v1/sites/{siteId}/topology", authService.RequireSite(auth.PermissionSitesRead, "siteId", siteHandler.Topology))
	mux.HandleFunc("POST /api/v1/reservations", authService.Require(auth.PermissionSessionsWrite, sessionHandler.CreateReservation))
	mux.HandleFunc("GET /api/v1/sessions", authService.Require(auth.PermissionSessionsRead, sessionHandler.List))
	mux.HandleFunc("GET /api/v1/sessions/{sessionId}", authService.Require(auth.PermissionSessionsRead, sessionHandler.Detail))
	mux.HandleFunc("POST /api/v1/sessions/{sessionId}/transition", authService.Require(auth.PermissionSessionsWrite, sessionHandler.Transition))
	mux.HandleFunc("POST /api/v1/sessions/{sessionId}/commands/start", authService.Require(auth.PermissionCommandsWrite, commandHandler.Create(command.TypeStart)))
	mux.HandleFunc("POST /api/v1/sessions/{sessionId}/commands/stop", authService.Require(auth.PermissionCommandsWrite, commandHandler.Create(command.TypeStop)))
	mux.HandleFunc("POST /api/v1/sessions/{sessionId}/commands/pause", authService.Require(auth.PermissionCommandsWrite, commandHandler.Create(command.TypePause)))
	mux.HandleFunc("POST /api/v1/sessions/{sessionId}/commands/resume", authService.Require(auth.PermissionCommandsWrite, commandHandler.Create(command.TypeResume)))
	mux.HandleFunc("POST /api/v1/sessions/{sessionId}/commands/reset", authService.Require(auth.PermissionCommandsWrite, commandHandler.Create(command.TypeReset)))
	mux.HandleFunc("POST /api/v1/sessions/{sessionId}/commands/limit-power", authService.Require(auth.PermissionCommandsWrite, commandHandler.Create(command.TypeLimitPower)))
	mux.HandleFunc("POST /api/v1/gateway/chargers/register", gatewayHandler.Register)
	mux.HandleFunc("POST /api/v1/gateway/chargers/{chargerCode}/heartbeat", gatewayHandler.Heartbeat)
	mux.HandleFunc("POST /api/v1/gateway/chargers/{chargerCode}/status", gatewayHandler.Status)
	mux.HandleFunc("POST /api/v1/gateway/chargers/{chargerCode}/meter-values", gatewayHandler.MeterValue)
	mux.HandleFunc("POST /api/v1/gateway/chargers/{chargerCode}/alarms", gatewayHandler.Alarm)
	mux.HandleFunc("POST /api/v1/gateway/chargers/{chargerCode}/command-receipts", commandHandler.Receipt)
	mux.HandleFunc("POST /api/v1/gateway/chargers/{chargerCode}/offline", gatewayHandler.Offline)
	mux.HandleFunc("GET /api/v1/pricing-policies", authService.Require(auth.PermissionBillingRead, billingHandler.ListPolicies))
	mux.HandleFunc("GET /api/v1/billing-drafts", authService.Require(auth.PermissionBillingRead, billingHandler.ListDrafts))
	mux.HandleFunc("POST /api/v1/sessions/{sessionId}/billing-draft", authService.Require(auth.PermissionBillingGenerate, billingHandler.GenerateDraft))
	mux.HandleFunc("GET /api/v1/reconciliation-exceptions", authService.Require(auth.PermissionBillingRead, billingHandler.ListExceptions))
	mux.HandleFunc("POST /api/v1/reconciliation-exceptions/{exceptionId}/review", authService.Require(auth.PermissionFinanceReview, financeHandler.ReviewException))
	mux.HandleFunc("POST /api/v1/reconciliation-exceptions/{exceptionId}/corrections", authService.Require(auth.PermissionFinanceReview, financeHandler.CreateCorrection))
	mux.HandleFunc("GET /api/v1/reconciliation-exceptions/export", authService.Require(auth.PermissionFinanceReview, financeHandler.ExportReconciliation))
	mux.HandleFunc("POST /api/v1/billing-drafts/{billId}/confirm", authService.Require(auth.PermissionFinanceReview, financeHandler.ConfirmBill))
	mux.HandleFunc("GET /api/v1/sites/{siteId}/load-control", authService.RequireSite(auth.PermissionLoadControlRead, "siteId", loadControlHandler.Snapshot))
	mux.HandleFunc("POST /api/v1/load-control/records", authService.Require(auth.PermissionLoadControl, loadControlHandler.CreateRecord))
	mux.HandleFunc("GET /api/v1/maintenance", authService.Require(auth.PermissionMaintenanceRead, maintenanceHandler.Snapshot))
	mux.HandleFunc("POST /api/v1/faults/{faultId}/work-order", authService.Require(auth.PermissionMaintenance, maintenanceHandler.CreateWorkOrder))
	mux.HandleFunc("GET /api/v1/work-orders/{workOrderId}/events", authService.Require(auth.PermissionMaintenanceRead, maintenanceHandler.WorkOrderEvents))
	mux.HandleFunc("POST /api/v1/work-orders/{workOrderId}/transition", authService.Require(auth.PermissionMaintenance, maintenanceHandler.TransitionWorkOrder))
	mux.HandleFunc("GET /api/v1/audit-logs", authService.Require(auth.PermissionAuditRead, auditHandler.List))
	mux.HandleFunc("GET /api/v1/config/sites/{siteId}", authService.Require(auth.PermissionConfigManage, configHandler.Snapshot))
	mux.HandleFunc("PATCH /api/v1/config/sites/{siteId}", authService.Require(auth.PermissionConfigManage, configHandler.UpdateSite))
	mux.HandleFunc("PATCH /api/v1/config/areas/{areaId}", authService.Require(auth.PermissionConfigManage, configHandler.UpdateArea))
	mux.HandleFunc("PATCH /api/v1/config/charger-groups/{groupId}", authService.Require(auth.PermissionConfigManage, configHandler.UpdateGroup))
	mux.HandleFunc("PATCH /api/v1/config/chargers/{chargerId}", authService.Require(auth.PermissionConfigManage, configHandler.UpdateCharger))
	mux.HandleFunc("PATCH /api/v1/config/connectors/{connectorId}", authService.Require(auth.PermissionConfigManage, configHandler.UpdateConnector))
	mux.HandleFunc("PATCH /api/v1/config/load-policies/{policyId}", authService.Require(auth.PermissionConfigManage, configHandler.UpdateLoadPolicy))
	mux.HandleFunc("PATCH /api/v1/config/reservation-rules/{ruleId}", authService.Require(auth.PermissionConfigManage, configHandler.UpdateReservationRule))
	mux.HandleFunc("PATCH /api/v1/config/queue-rules/{ruleId}", authService.Require(auth.PermissionConfigManage, configHandler.UpdateQueueRule))
	mux.HandleFunc("POST /api/v1/config/pricing-policies", authService.Require(auth.PermissionConfigManage, configHandler.CreatePricingPolicy))
	mux.HandleFunc("POST /api/v1/ai/session-explanations", authService.Require(auth.PermissionAIRead, aiHandler.ExplainSession))
	mux.HandleFunc("POST /api/v1/ai/work-order-summaries", authService.Require(auth.PermissionAIRead, aiHandler.SummarizeWorkOrder))
	mux.HandleFunc("POST /api/v1/ai/congestion-risk", authService.Require(auth.PermissionAIRead, aiHandler.PredictCongestion))
	mux.HandleFunc("POST /api/v1/ai/daily-reports", authService.Require(auth.PermissionAIRead, aiHandler.DraftDailyReport))
	mux.HandleFunc("POST /api/v1/ai/station-qa", authService.Require(auth.PermissionAIRead, aiHandler.AnswerStationQuestion))
	mux.HandleFunc("GET /api/v1/simulator/status", authService.Require(auth.PermissionSimulator, simulatorHandler.Status))
	mux.HandleFunc("POST /api/v1/simulator/once", authService.Require(auth.PermissionSimulator, simulatorHandler.RunOnce))
	mux.HandleFunc("POST /api/v1/simulator/start", authService.Require(auth.PermissionSimulator, simulatorHandler.Start))
	mux.HandleFunc("POST /api/v1/simulator/stop", authService.Require(auth.PermissionSimulator, simulatorHandler.Stop))
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
