package loadcontrol

import (
	"encoding/json"
	"errors"
	"net/http"

	"charging-ops/backend/internal/common/response"
	"charging-ops/backend/internal/domain/auth"
)

// Handler serves load-control APIs.
type Handler struct {
	service *Service
}

// NewHandler creates a load-control HTTP handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Snapshot returns load policies, queue, current load, and control records.
func (h *Handler) Snapshot(w http.ResponseWriter, r *http.Request) {
	snapshot, err := h.service.Snapshot(r.Context(), r.PathValue("siteId"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	response.OK(w, r, snapshot)
}

// CreateRecord stores a manual control decision.
func (h *Handler) CreateRecord(w http.ResponseWriter, r *http.Request) {
	var request createRecordRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		response.Error(w, r, http.StatusBadRequest, 10002, "请求格式错误", "invalid json body")
		return
	}

	record, err := h.service.CreateRecord(r.Context(), CreateRecordParams{
		SiteID:        request.SiteID,
		SessionID:     request.SessionID,
		ActionType:    request.ActionType,
		Reason:        request.Reason,
		BeforeLoadKW:  request.BeforeLoadKW,
		AfterLoadKW:   request.AfterLoadKW,
		TargetPowerKW: request.TargetPowerKW,
		Status:        request.Status,
		OperatorName:  request.OperatorName,
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	response.OK(w, r, record)
}

type createRecordRequest struct {
	SiteID        string   `json:"siteId"`
	SessionID     string   `json:"sessionId"`
	ActionType    string   `json:"actionType"`
	Reason        string   `json:"reason"`
	BeforeLoadKW  float64  `json:"beforeLoadKw"`
	AfterLoadKW   float64  `json:"afterLoadKw"`
	TargetPowerKW *float64 `json:"targetPowerKw"`
	Status        string   `json:"status"`
	OperatorName  string   `json:"operatorName"`
}

func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		response.Error(w, r, http.StatusNotFound, 30001, "负载控制对象不存在", "load-control object not found")
	case errors.Is(err, ErrInvalidRecord):
		response.Error(w, r, http.StatusBadRequest, 30030, "负载控制记录不合法", "invalid load-control record")
	case errors.Is(err, auth.ErrForbidden):
		response.Error(w, r, http.StatusForbidden, 20002, "权限不足", "permission denied")
	default:
		response.Error(w, r, http.StatusInternalServerError, 50002, "负载控制处理失败", "load-control operation failed")
	}
}
