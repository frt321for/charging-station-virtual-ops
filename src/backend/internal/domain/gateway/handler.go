package gateway

import (
	"encoding/json"
	"errors"
	"net/http"

	"charging-ops/backend/internal/common/response"
)

// Handler serves virtual charger gateway APIs.
type Handler struct {
	service *Service
}

// NewHandler creates a virtual charger gateway handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Register handles virtual charger registration.
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var request registerRequest
	if err := decodeJSON(r, &request); err != nil {
		response.Error(w, r, http.StatusBadRequest, 10002, "请求格式错误", "invalid json body")
		return
	}

	snapshot, err := h.service.Register(r.Context(), RegisterRequest(request))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	response.OK(w, r, snapshot)
}

// Heartbeat handles virtual charger heartbeat reports.
func (h *Handler) Heartbeat(w http.ResponseWriter, r *http.Request) {
	var request heartbeatRequest
	if err := decodeJSON(r, &request); err != nil {
		response.Error(w, r, http.StatusBadRequest, 10002, "请求格式错误", "invalid json body")
		return
	}
	snapshot, err := h.service.Heartbeat(r.Context(), r.PathValue("chargerCode"), HeartbeatRequest(request))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	response.OK(w, r, snapshot)
}

// Status handles charger and connector status reports.
func (h *Handler) Status(w http.ResponseWriter, r *http.Request) {
	var request statusRequest
	if err := decodeJSON(r, &request); err != nil {
		response.Error(w, r, http.StatusBadRequest, 10002, "请求格式错误", "invalid json body")
		return
	}
	snapshot, err := h.service.Status(r.Context(), r.PathValue("chargerCode"), StatusRequest(request))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	response.OK(w, r, snapshot)
}

// MeterValue handles charger meter value reports.
func (h *Handler) MeterValue(w http.ResponseWriter, r *http.Request) {
	var request MeterValueRequest
	if err := decodeJSON(r, &request); err != nil {
		response.Error(w, r, http.StatusBadRequest, 10002, "请求格式错误", "invalid json body")
		return
	}
	snapshot, err := h.service.MeterValue(r.Context(), r.PathValue("chargerCode"), request)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	response.OK(w, r, snapshot)
}

// Alarm handles charger fault alarms.
func (h *Handler) Alarm(w http.ResponseWriter, r *http.Request) {
	var request AlarmRequest
	if err := decodeJSON(r, &request); err != nil {
		response.Error(w, r, http.StatusBadRequest, 10002, "请求格式错误", "invalid json body")
		return
	}
	snapshot, err := h.service.Alarm(r.Context(), r.PathValue("chargerCode"), request)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	response.OK(w, r, snapshot)
}

// Offline handles charger offline reports.
func (h *Handler) Offline(w http.ResponseWriter, r *http.Request) {
	snapshot, err := h.service.Offline(r.Context(), r.PathValue("chargerCode"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	response.OK(w, r, snapshot)
}

type registerRequest RegisterRequest
type heartbeatRequest HeartbeatRequest
type statusRequest StatusRequest

func decodeJSON(r *http.Request, target any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}

func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		response.Error(w, r, http.StatusNotFound, 30001, "桩机对象不存在", "gateway target not found")
	case errors.Is(err, ErrInvalidRequest):
		response.Error(w, r, http.StatusBadRequest, 10002, "桩机上报参数不合法", "invalid gateway request")
	default:
		response.Error(w, r, http.StatusInternalServerError, 50002, "桩机上报处理失败", "gateway operation failed")
	}
}
