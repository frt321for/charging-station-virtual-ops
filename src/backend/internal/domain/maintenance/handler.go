package maintenance

import (
	"encoding/json"
	"errors"
	"net/http"

	"charging-ops/backend/internal/common/response"
	"charging-ops/backend/internal/domain/auth"
)

// Handler serves maintenance and SLA APIs.
type Handler struct {
	service *Service
}

// NewHandler creates a maintenance HTTP handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Snapshot returns faults, work orders, and SLA counters.
func (h *Handler) Snapshot(w http.ResponseWriter, r *http.Request) {
	snapshot, err := h.service.Snapshot(r.Context(), r.URL.Query().Get("siteId"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	response.OK(w, r, snapshot)
}

// WorkOrderEvents returns a ticket timeline.
func (h *Handler) WorkOrderEvents(w http.ResponseWriter, r *http.Request) {
	events, err := h.service.ListEvents(r.Context(), r.PathValue("workOrderId"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	response.OK(w, r, map[string]any{"list": events})
}

// CreateWorkOrder creates or links a work order for a fault.
func (h *Handler) CreateWorkOrder(w http.ResponseWriter, r *http.Request) {
	var request createWorkOrderRequest
	if err := decodeJSON(r, &request); err != nil {
		response.Error(w, r, http.StatusBadRequest, 10002, "请求格式错误", "invalid json body")
		return
	}

	workOrder, err := h.service.CreateWorkOrder(r.Context(), CreateWorkOrderParams{
		FaultID:      r.PathValue("faultId"),
		AssigneeName: request.AssigneeName,
		ImpactScope:  request.ImpactScope,
		Title:        request.Title,
		Description:  request.Description,
		ActorName:    request.ActorName,
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	response.OK(w, r, workOrder)
}

// TransitionWorkOrder advances a work-order state.
func (h *Handler) TransitionWorkOrder(w http.ResponseWriter, r *http.Request) {
	var request transitionRequest
	if err := decodeJSON(r, &request); err != nil {
		response.Error(w, r, http.StatusBadRequest, 10002, "请求格式错误", "invalid json body")
		return
	}

	workOrder, err := h.service.TransitionWorkOrder(r.Context(), TransitionParams{
		WorkOrderID:  r.PathValue("workOrderId"),
		TargetStatus: request.TargetStatus,
		AssigneeName: request.AssigneeName,
		ActorName:    request.ActorName,
		Note:         request.Note,
		Payload:      request.Payload,
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	response.OK(w, r, workOrder)
}

type createWorkOrderRequest struct {
	AssigneeName string `json:"assigneeName"`
	ImpactScope  string `json:"impactScope"`
	Title        string `json:"title"`
	Description  string `json:"description"`
	ActorName    string `json:"actorName"`
}

type transitionRequest struct {
	TargetStatus string          `json:"targetStatus"`
	AssigneeName string          `json:"assigneeName"`
	ActorName    string          `json:"actorName"`
	Note         string          `json:"note"`
	Payload      json.RawMessage `json:"payload"`
}

func decodeJSON(r *http.Request, target any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}

func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		response.Error(w, r, http.StatusNotFound, 30001, "维保对象不存在", "maintenance object not found")
	case errors.Is(err, ErrInvalidWorkOrder):
		response.Error(w, r, http.StatusBadRequest, 30040, "工单流转不合法", "invalid work-order request")
	case errors.Is(err, auth.ErrForbidden):
		response.Error(w, r, http.StatusForbidden, 20002, "权限不足", "permission denied")
	default:
		response.Error(w, r, http.StatusInternalServerError, 50002, "维保处理失败", "maintenance operation failed")
	}
}
