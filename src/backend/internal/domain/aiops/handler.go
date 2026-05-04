package aiops

import (
	"encoding/json"
	"errors"
	"net/http"

	"charging-ops/backend/internal/common/response"
	"charging-ops/backend/internal/domain/auth"
)

// Handler serves read-only AI operations assistant APIs.
type Handler struct {
	service *Service
}

// NewHandler creates an AI operations HTTP handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// ExplainSession returns an abnormal session explanation.
func (h *Handler) ExplainSession(w http.ResponseWriter, r *http.Request) {
	var request sessionRequest
	if err := decodeJSON(r, &request); err != nil {
		response.Error(w, r, http.StatusBadRequest, 10002, "请求格式错误", "invalid json body")
		return
	}
	insight, err := h.service.ExplainSession(r.Context(), SessionExplanationParams{SessionID: request.SessionID})
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	response.OK(w, r, insight)
}

// SummarizeWorkOrder returns a work-order handoff summary.
func (h *Handler) SummarizeWorkOrder(w http.ResponseWriter, r *http.Request) {
	var request workOrderRequest
	if err := decodeJSON(r, &request); err != nil {
		response.Error(w, r, http.StatusBadRequest, 10002, "请求格式错误", "invalid json body")
		return
	}
	insight, err := h.service.SummarizeWorkOrder(r.Context(), WorkOrderSummaryParams{WorkOrderID: request.WorkOrderID})
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	response.OK(w, r, insight)
}

// PredictCongestion returns a station congestion risk note.
func (h *Handler) PredictCongestion(w http.ResponseWriter, r *http.Request) {
	var request siteRiskRequest
	if err := decodeJSON(r, &request); err != nil {
		response.Error(w, r, http.StatusBadRequest, 10002, "请求格式错误", "invalid json body")
		return
	}
	insight, err := h.service.PredictCongestion(r.Context(), CongestionRiskParams{
		SiteID:       request.SiteID,
		HorizonHours: request.HorizonHours,
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	response.OK(w, r, insight)
}

// DraftDailyReport returns an editable station daily report draft.
func (h *Handler) DraftDailyReport(w http.ResponseWriter, r *http.Request) {
	var request dailyReportRequest
	if err := decodeJSON(r, &request); err != nil {
		response.Error(w, r, http.StatusBadRequest, 10002, "请求格式错误", "invalid json body")
		return
	}
	insight, err := h.service.DraftDailyReport(r.Context(), DailyReportParams{
		SiteID:       request.SiteID,
		BusinessDate: request.BusinessDate,
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	response.OK(w, r, insight)
}

// AnswerStationQuestion answers a bounded station question.
func (h *Handler) AnswerStationQuestion(w http.ResponseWriter, r *http.Request) {
	var request stationQARequest
	if err := decodeJSON(r, &request); err != nil {
		response.Error(w, r, http.StatusBadRequest, 10002, "请求格式错误", "invalid json body")
		return
	}
	insight, err := h.service.AnswerStationQuestion(r.Context(), StationQAParams{
		SiteID:   request.SiteID,
		Question: request.Question,
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	response.OK(w, r, insight)
}

type sessionRequest struct {
	SessionID string `json:"sessionId"`
}

type workOrderRequest struct {
	WorkOrderID string `json:"workOrderId"`
}

type siteRiskRequest struct {
	SiteID       string `json:"siteId"`
	HorizonHours int    `json:"horizonHours"`
}

type dailyReportRequest struct {
	SiteID       string `json:"siteId"`
	BusinessDate string `json:"businessDate"`
}

type stationQARequest struct {
	SiteID   string `json:"siteId"`
	Question string `json:"question"`
}

func decodeJSON(r *http.Request, target any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}

func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		response.Error(w, r, http.StatusNotFound, 30001, "AI 对象不存在", "ai context target not found")
	case errors.Is(err, ErrInvalidRequest):
		response.Error(w, r, http.StatusBadRequest, 30060, "AI 请求不完整", "invalid ai request")
	case errors.Is(err, auth.ErrForbidden):
		response.Error(w, r, http.StatusForbidden, 20002, "权限不足", "permission denied")
	default:
		response.Error(w, r, http.StatusInternalServerError, 50002, "AI 助手处理失败", "ai assistant operation failed")
	}
}
