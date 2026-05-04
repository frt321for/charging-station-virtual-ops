package configmgmt

import (
	"encoding/json"
	"errors"
	"net/http"

	"charging-ops/backend/internal/common/response"
)

// Handler serves configuration management APIs.
type Handler struct {
	service *Service
}

// NewHandler creates a configuration handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Snapshot returns editable station configuration.
func (h *Handler) Snapshot(w http.ResponseWriter, r *http.Request) {
	snapshot, err := h.service.Snapshot(r.Context(), r.PathValue("siteId"))
	if err != nil {
		writeConfigError(w, r, err)
		return
	}
	response.OK(w, r, snapshot)
}

// UpdateSite updates station-level settings.
func (h *Handler) UpdateSite(w http.ResponseWriter, r *http.Request) {
	var request SiteUpdate
	if !decodeConfigRequest(w, r, &request) {
		return
	}
	result, err := h.service.UpdateSite(r.Context(), r.PathValue("siteId"), request)
	if err != nil {
		writeConfigError(w, r, err)
		return
	}
	response.OK(w, r, result)
}

func (h *Handler) UpdateArea(w http.ResponseWriter, r *http.Request) {
	var request AreaUpdate
	if !decodeConfigRequest(w, r, &request) {
		return
	}
	result, err := h.service.UpdateArea(r.Context(), r.PathValue("areaId"), request)
	if err != nil {
		writeConfigError(w, r, err)
		return
	}
	response.OK(w, r, result)
}

func (h *Handler) UpdateGroup(w http.ResponseWriter, r *http.Request) {
	var request GroupUpdate
	if !decodeConfigRequest(w, r, &request) {
		return
	}
	result, err := h.service.UpdateGroup(r.Context(), r.PathValue("groupId"), request)
	if err != nil {
		writeConfigError(w, r, err)
		return
	}
	response.OK(w, r, result)
}

func (h *Handler) UpdateCharger(w http.ResponseWriter, r *http.Request) {
	var request ChargerUpdate
	if !decodeConfigRequest(w, r, &request) {
		return
	}
	result, err := h.service.UpdateCharger(r.Context(), r.PathValue("chargerId"), request)
	if err != nil {
		writeConfigError(w, r, err)
		return
	}
	response.OK(w, r, result)
}

func (h *Handler) UpdateConnector(w http.ResponseWriter, r *http.Request) {
	var request ConnectorUpdate
	if !decodeConfigRequest(w, r, &request) {
		return
	}
	result, err := h.service.UpdateConnector(r.Context(), r.PathValue("connectorId"), request)
	if err != nil {
		writeConfigError(w, r, err)
		return
	}
	response.OK(w, r, result)
}

func (h *Handler) UpdateLoadPolicy(w http.ResponseWriter, r *http.Request) {
	var request LoadPolicyUpdate
	if !decodeConfigRequest(w, r, &request) {
		return
	}
	result, err := h.service.UpdateLoadPolicy(r.Context(), r.PathValue("policyId"), request)
	if err != nil {
		writeConfigError(w, r, err)
		return
	}
	response.OK(w, r, result)
}

func (h *Handler) UpdateReservationRule(w http.ResponseWriter, r *http.Request) {
	var request ReservationRuleUpdate
	if !decodeConfigRequest(w, r, &request) {
		return
	}
	result, err := h.service.UpdateReservationRule(r.Context(), r.PathValue("ruleId"), request)
	if err != nil {
		writeConfigError(w, r, err)
		return
	}
	response.OK(w, r, result)
}

func (h *Handler) UpdateQueueRule(w http.ResponseWriter, r *http.Request) {
	var request QueueRuleUpdate
	if !decodeConfigRequest(w, r, &request) {
		return
	}
	result, err := h.service.UpdateQueueRule(r.Context(), r.PathValue("ruleId"), request)
	if err != nil {
		writeConfigError(w, r, err)
		return
	}
	response.OK(w, r, result)
}

// CreatePricingPolicy creates a new versioned pricing policy.
func (h *Handler) CreatePricingPolicy(w http.ResponseWriter, r *http.Request) {
	var request PricingPolicyCreate
	if !decodeConfigRequest(w, r, &request) {
		return
	}
	result, err := h.service.CreatePricingPolicy(r.Context(), request)
	if err != nil {
		writeConfigError(w, r, err)
		return
	}
	response.OK(w, r, result)
}

func decodeConfigRequest(w http.ResponseWriter, r *http.Request, target any) bool {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		response.Error(w, r, http.StatusBadRequest, 10002, "配置请求不合法", "invalid config request")
		return false
	}
	return true
}

func writeConfigError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrForbidden):
		response.Error(w, r, http.StatusForbidden, 20002, "权限不足", "site boundary denied")
	case errors.Is(err, ErrInvalidRequest):
		response.Error(w, r, http.StatusBadRequest, 10002, "配置请求不合法", "invalid config request")
	case errors.Is(err, ErrNotFound):
		response.Error(w, r, http.StatusNotFound, 30001, "配置对象不存在", "config target not found")
	default:
		response.Error(w, r, http.StatusInternalServerError, 50002, "配置更新失败", "config operation failed")
	}
}
