package finance

import (
	"encoding/json"
	"errors"
	"net/http"

	"charging-ops/backend/internal/common/response"
	"charging-ops/backend/internal/domain/auth"
)

// Handler serves finance reconciliation APIs.
type Handler struct {
	service *Service
}

// NewHandler creates a finance handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// ReviewException updates an exception review state.
func (h *Handler) ReviewException(w http.ResponseWriter, r *http.Request) {
	var request reviewRequest
	if err := decodeJSON(r, &request); err != nil {
		response.Error(w, r, http.StatusBadRequest, 10002, "请求格式错误", "invalid json body")
		return
	}
	err := h.service.ReviewException(r.Context(), ReviewParams{
		ExceptionID:  r.PathValue("exceptionId"),
		Status:       request.Status,
		ReviewerName: request.ReviewerName,
		Note:         request.Note,
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	response.OK(w, r, map[string]any{"status": request.Status})
}

// CreateCorrection stores a correction record.
func (h *Handler) CreateCorrection(w http.ResponseWriter, r *http.Request) {
	var request correctionRequest
	if err := decodeJSON(r, &request); err != nil {
		response.Error(w, r, http.StatusBadRequest, 10002, "请求格式错误", "invalid json body")
		return
	}
	correction, err := h.service.CreateCorrection(r.Context(), CorrectionParams{
		ExceptionID:              r.PathValue("exceptionId"),
		CorrectedEnergyKWh:       request.CorrectedEnergyKWh,
		CorrectedDurationMinutes: request.CorrectedDurationMinutes,
		CorrectedTotalAmount:     request.CorrectedTotalAmount,
		Reason:                   request.Reason,
		ReviewerName:             request.ReviewerName,
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	response.OK(w, r, correction)
}

// ConfirmBill confirms a billing draft.
func (h *Handler) ConfirmBill(w http.ResponseWriter, r *http.Request) {
	var request confirmRequest
	if err := decodeJSON(r, &request); err != nil {
		response.Error(w, r, http.StatusBadRequest, 10002, "请求格式错误", "invalid json body")
		return
	}
	err := h.service.ConfirmBill(r.Context(), ConfirmParams{
		BillID:       r.PathValue("billId"),
		ReviewerName: request.ReviewerName,
		Note:         request.Note,
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	response.OK(w, r, map[string]any{"status": "confirmed"})
}

// ExportReconciliation returns a reconciliation export package.
func (h *Handler) ExportReconciliation(w http.ResponseWriter, r *http.Request) {
	export, err := h.service.ExportReconciliation(
		r.Context(),
		r.URL.Query().Get("siteId"),
		r.URL.Query().Get("generatedBy"),
	)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	response.OK(w, r, export)
}

type reviewRequest struct {
	Status       string `json:"status"`
	ReviewerName string `json:"reviewerName"`
	Note         string `json:"note"`
}

type correctionRequest struct {
	CorrectedEnergyKWh       *float64 `json:"correctedEnergyKwh"`
	CorrectedDurationMinutes *int     `json:"correctedDurationMinutes"`
	CorrectedTotalAmount     *float64 `json:"correctedTotalAmount"`
	Reason                   string   `json:"reason"`
	ReviewerName             string   `json:"reviewerName"`
}

type confirmRequest struct {
	ReviewerName string `json:"reviewerName"`
	Note         string `json:"note"`
}

func decodeJSON(r *http.Request, target any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}

func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		response.Error(w, r, http.StatusNotFound, 30001, "财务对象不存在", "finance object not found")
	case errors.Is(err, ErrInvalidReview):
		response.Error(w, r, http.StatusBadRequest, 30050, "核查处理不合法", "invalid reconciliation review")
	case errors.Is(err, auth.ErrForbidden):
		response.Error(w, r, http.StatusForbidden, 20002, "权限不足", "permission denied")
	default:
		response.Error(w, r, http.StatusInternalServerError, 50002, "财务处理失败", "finance operation failed")
	}
}
