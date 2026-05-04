package command

import (
	"encoding/json"
	"errors"
	"net/http"

	"charging-ops/backend/internal/common/response"
)

// Handler serves remote command APIs.
type Handler struct {
	service *Service
}

// NewHandler creates a remote command handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Create records a command sent from operations to a charger.
func (h *Handler) Create(commandType Type) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request createCommandRequest
		if err := decodeJSON(r, &request); err != nil {
			response.Error(w, r, http.StatusBadRequest, 10002, "请求格式错误", "invalid json body")
			return
		}

		command, err := h.service.Create(r.Context(), CreateRequest{
			SessionID:     r.PathValue("sessionId"),
			CommandType:   commandType,
			RequestedBy:   request.RequestedBy,
			TargetPowerKW: request.TargetPowerKW,
			Payload:       request.Payload,
		})
		if err != nil {
			h.writeError(w, r, err)
			return
		}

		response.OK(w, r, command)
	}
}

// Receipt records a command receipt reported by a virtual charger.
func (h *Handler) Receipt(w http.ResponseWriter, r *http.Request) {
	var request receiptRequest
	if err := decodeJSON(r, &request); err != nil {
		response.Error(w, r, http.StatusBadRequest, 10002, "请求格式错误", "invalid json body")
		return
	}

	command, err := h.service.ApplyReceipt(r.Context(), ReceiptRequest{
		ChargerCode: r.PathValue("chargerCode"),
		CommandNo:   request.CommandNo,
		SessionNo:   request.SessionNo,
		CommandType: request.CommandType,
		Receipt:     request.Receipt,
		Message:     request.Message,
		Payload:     request.Payload,
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	response.OK(w, r, command)
}

type createCommandRequest struct {
	RequestedBy   string          `json:"requestedBy"`
	TargetPowerKW *float64        `json:"targetPowerKw"`
	Payload       json.RawMessage `json:"payload"`
}

type receiptRequest struct {
	CommandNo   string          `json:"commandNo"`
	SessionNo   string          `json:"sessionNo"`
	CommandType Type            `json:"commandType"`
	Receipt     Status          `json:"receipt"`
	Message     string          `json:"message"`
	Payload     json.RawMessage `json:"payload"`
}

func decodeJSON(r *http.Request, target any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}

func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		response.Error(w, r, http.StatusNotFound, 30001, "命令对象不存在", "command target not found")
	case errors.Is(err, ErrChargerOffline):
		response.Error(w, r, http.StatusConflict, 40003, "桩机离线，不能下发命令", "charger offline")
	case errors.Is(err, ErrInvalidCommand), errors.Is(err, ErrInvalidReceipt):
		response.Error(w, r, http.StatusBadRequest, 10002, "命令参数不合法", "invalid command request")
	case errors.Is(err, ErrInvalidTransition):
		response.Error(w, r, http.StatusBadRequest, 30011, "会话状态流转不允许", "invalid command transition")
	default:
		response.Error(w, r, http.StatusInternalServerError, 50002, "命令处理失败", "command operation failed")
	}
}
