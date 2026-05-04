package session

import (
	"encoding/json"
	"errors"
	"net/http"

	"charging-ops/backend/internal/common/response"
	"charging-ops/backend/internal/domain/auth"
)

// Handler serves charging session APIs.
type Handler struct {
	service *Service
}

// NewHandler creates a charging session HTTP handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// List returns recent charging sessions.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	sessions, err := h.service.List(r.Context())
	if err != nil {
		if errors.Is(err, auth.ErrForbidden) {
			response.Error(w, r, http.StatusForbidden, 20002, "权限不足", "permission denied")
			return
		}
		response.Error(w, r, http.StatusInternalServerError, 50002, "会话列表查询失败", "query failed")
		return
	}

	response.OK(w, r, map[string]any{
		"list": sessions,
	})
}

// Detail returns one charging session with timeline and meter values.
func (h *Handler) Detail(w http.ResponseWriter, r *http.Request) {
	detail, err := h.service.Get(r.Context(), r.PathValue("sessionId"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	response.OK(w, r, detail)
}

// CreateReservation creates a reservation-backed session.
func (h *Handler) CreateReservation(w http.ResponseWriter, r *http.Request) {
	var request createReservationRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		response.Error(w, r, http.StatusBadRequest, 10002, "请求格式错误", "invalid json body")
		return
	}

	detail, err := h.service.CreateReservation(r.Context(), CreateReservationParams{
		ConnectorCode:      request.ConnectorCode,
		ReservationMinutes: request.ReservationMinutes,
		RequestedBy:        request.RequestedBy,
		Payload:            request.Payload,
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	response.OK(w, r, detail)
}

// Transition moves a charging session through the documented lifecycle.
func (h *Handler) Transition(w http.ResponseWriter, r *http.Request) {
	var request transitionRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		response.Error(w, r, http.StatusBadRequest, 10002, "请求格式错误", "invalid json body")
		return
	}
	if request.TargetStatus == "" {
		response.Error(w, r, http.StatusBadRequest, 10001, "目标状态不能为空", "missing targetStatus")
		return
	}

	detail, err := h.service.Transition(r.Context(), TransitionParams{
		SessionID:    r.PathValue("sessionId"),
		TargetStatus: request.TargetStatus,
		EventType:    request.EventType,
		Source:       request.Source,
		Payload:      request.Payload,
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	response.OK(w, r, detail)
}

type transitionRequest struct {
	TargetStatus Status          `json:"targetStatus"`
	EventType    string          `json:"eventType"`
	Source       string          `json:"source"`
	Payload      json.RawMessage `json:"payload"`
}

type createReservationRequest struct {
	ConnectorCode      string          `json:"connectorCode"`
	ReservationMinutes int             `json:"reservationMinutes"`
	RequestedBy        string          `json:"requestedBy"`
	Payload            json.RawMessage `json:"payload"`
}

func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		response.Error(w, r, http.StatusNotFound, 30001, "会话不存在", "session not found")
	case errors.Is(err, ErrInvalidReservation):
		response.Error(w, r, http.StatusBadRequest, 30012, "预约请求不合法", "invalid reservation")
	case errors.Is(err, ErrInvalidStatus):
		response.Error(w, r, http.StatusBadRequest, 30010, "会话状态不存在", "invalid session status")
	case errors.Is(err, ErrInvalidTransition):
		response.Error(w, r, http.StatusBadRequest, 30011, "会话状态流转不允许", "invalid session transition")
	case errors.Is(err, auth.ErrForbidden):
		response.Error(w, r, http.StatusForbidden, 20002, "权限不足", "permission denied")
	default:
		response.Error(w, r, http.StatusInternalServerError, 50002, "会话处理失败", "session operation failed")
	}
}
