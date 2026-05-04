package audit

import (
	"net/http"
	"strconv"

	"charging-ops/backend/internal/common/response"
)

// Handler serves audit APIs.
type Handler struct {
	service *Service
}

// NewHandler creates an audit handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// List returns audit logs with simple filters.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	page, err := h.service.List(r.Context(), ListParams{
		EntityType: r.URL.Query().Get("entityType"),
		Action:     r.URL.Query().Get("action"),
		ActorName:  r.URL.Query().Get("actorName"),
		Page:       intQuery(r, "page"),
		PageSize:   intQuery(r, "pageSize"),
	})
	if err != nil {
		response.Error(w, r, http.StatusInternalServerError, 50002, "审计日志查询失败", "audit query failed")
		return
	}
	response.OK(w, r, page)
}

func intQuery(r *http.Request, key string) int {
	value := r.URL.Query().Get(key)
	if value == "" {
		return 0
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return parsed
}
