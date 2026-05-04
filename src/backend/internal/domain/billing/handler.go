package billing

import (
	"encoding/json"
	"errors"
	"net/http"

	"charging-ops/backend/internal/common/response"
	"charging-ops/backend/internal/domain/auth"
)

// Handler serves pricing, billing, and reconciliation APIs.
type Handler struct {
	service *Service
}

// NewHandler creates a billing HTTP handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// ListPolicies returns active and historical pricing policies.
func (h *Handler) ListPolicies(w http.ResponseWriter, r *http.Request) {
	policies, err := h.service.ListPolicies(r.Context(), r.URL.Query().Get("siteId"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	response.OK(w, r, map[string]any{"list": policies})
}

// ListDrafts returns recent billing drafts.
func (h *Handler) ListDrafts(w http.ResponseWriter, r *http.Request) {
	drafts, err := h.service.ListDrafts(r.Context(), r.URL.Query().Get("siteId"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	response.OK(w, r, map[string]any{"list": drafts})
}

// GenerateDraft creates or refreshes a billing draft for a stopped session.
func (h *Handler) GenerateDraft(w http.ResponseWriter, r *http.Request) {
	var request generateDraftRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		response.Error(w, r, http.StatusBadRequest, 10002, "请求格式错误", "invalid json body")
		return
	}

	draft, err := h.service.GenerateDraft(r.Context(), GenerateDraftParams{
		SessionID:   r.PathValue("sessionId"),
		GeneratedBy: request.GeneratedBy,
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	response.OK(w, r, draft)
}

// ListExceptions returns reconciliation exceptions.
func (h *Handler) ListExceptions(w http.ResponseWriter, r *http.Request) {
	exceptions, err := h.service.ListExceptions(r.Context(), r.URL.Query().Get("siteId"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	response.OK(w, r, map[string]any{"list": exceptions})
}

type generateDraftRequest struct {
	GeneratedBy string `json:"generatedBy"`
}

func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		response.Error(w, r, http.StatusNotFound, 30001, "计费对象不存在", "billing object not found")
	case errors.Is(err, ErrInvalidDraft):
		response.Error(w, r, http.StatusBadRequest, 30020, "账单草稿无法生成", "invalid billing draft")
	case errors.Is(err, auth.ErrForbidden):
		response.Error(w, r, http.StatusForbidden, 20002, "权限不足", "permission denied")
	default:
		response.Error(w, r, http.StatusInternalServerError, 50002, "计费处理失败", "billing operation failed")
	}
}
