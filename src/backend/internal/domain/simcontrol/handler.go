package simcontrol

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"charging-ops/backend/internal/common/response"
	"charging-ops/backend/internal/domain/auth"
)

// Handler serves browser-facing simulator control APIs.
type Handler struct {
	service *Service
}

// NewHandler creates a simulator control handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Status returns the simulator controller status.
func (h *Handler) Status(w http.ResponseWriter, r *http.Request) {
	response.OK(w, r, h.service.Status())
}

// RunOnce starts one simulator lifecycle and waits for completion.
func (h *Handler) RunOnce(w http.ResponseWriter, r *http.Request) {
	request, ok := decodeRequest(w, r)
	if !ok {
		return
	}
	status, err := h.service.RunOnce(r.Context(), request, authorizationFromRequest(r))
	if err != nil {
		writeError(w, r, err)
		return
	}
	response.OK(w, r, status)
}

// Start begins continuous simulator heartbeats.
func (h *Handler) Start(w http.ResponseWriter, r *http.Request) {
	request, ok := decodeRequest(w, r)
	if !ok {
		return
	}
	status, err := h.service.Start(request, authorizationFromRequest(r))
	if err != nil {
		writeError(w, r, err)
		return
	}
	response.OK(w, r, status)
}

// Stop cancels continuous simulator heartbeats.
func (h *Handler) Stop(w http.ResponseWriter, r *http.Request) {
	response.OK(w, r, h.service.Stop())
}

func decodeRequest(w http.ResponseWriter, r *http.Request) (Request, bool) {
	var request Request
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		response.Error(w, r, http.StatusBadRequest, 10002, "请求格式错误", "invalid json body")
		return Request{}, false
	}
	return request, true
}

func writeError(w http.ResponseWriter, r *http.Request, err error) {
	message := err.Error()
	switch {
	case strings.Contains(message, "already running"):
		response.Error(w, r, http.StatusConflict, 40011, "模拟器已经运行", "simulator already running")
	case errors.Is(err, context.Canceled):
		response.Error(w, r, http.StatusBadRequest, 10002, "模拟器请求已取消", "simulator request canceled")
	case strings.Contains(message, "must be"):
		response.Error(w, r, http.StatusBadRequest, 10002, "模拟器参数不合法", message)
	default:
		response.Error(w, r, http.StatusInternalServerError, 50002, "模拟器运行失败", message)
	}
}

func authorizationFromRequest(r *http.Request) string {
	token, ok := auth.TokenFromContext(r.Context())
	if !ok {
		return ""
	}
	return "Bearer " + token
}
