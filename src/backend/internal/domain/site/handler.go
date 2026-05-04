package site

import (
	"errors"
	"net/http"

	"charging-ops/backend/internal/common/response"
	"charging-ops/backend/internal/domain/auth"
)

// Handler serves site operations APIs.
type Handler struct {
	service *Service
}

// NewHandler creates a site HTTP handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// List returns site operation summaries.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	sites, err := h.service.ListSummaries(r.Context())
	if err != nil {
		if errors.Is(err, auth.ErrForbidden) {
			response.Error(w, r, http.StatusForbidden, 20002, "权限不足", "permission denied")
			return
		}
		response.Error(w, r, http.StatusInternalServerError, 50002, "站点列表查询失败", "query failed")
		return
	}

	response.OK(w, r, map[string]any{
		"list": sites,
	})
}

// Topology returns the nested charger topology for one site.
func (h *Handler) Topology(w http.ResponseWriter, r *http.Request) {
	siteID := r.PathValue("siteId")
	topology, err := h.service.GetTopology(r.Context(), siteID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			response.Error(w, r, http.StatusNotFound, 30001, "站点不存在", "site not found")
			return
		}
		if errors.Is(err, auth.ErrForbidden) {
			response.Error(w, r, http.StatusForbidden, 20002, "权限不足", "permission denied")
			return
		}
		response.Error(w, r, http.StatusInternalServerError, 50002, "站点拓扑查询失败", "query failed")
		return
	}

	response.OK(w, r, topology)
}
