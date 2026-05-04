package finance

import (
	"context"
	"fmt"
	"strings"
)

type repository interface {
	ReviewException(ctx context.Context, params ReviewParams) error
	CreateCorrection(ctx context.Context, params CorrectionParams) (Correction, error)
	ConfirmBill(ctx context.Context, params ConfirmParams) error
	ExportReconciliation(ctx context.Context, siteID string, generatedBy string) (ReconciliationExport, error)
}

// Service coordinates finance review, correction, confirmation, and export.
type Service struct {
	repository repository
}

// NewService creates a finance service.
func NewService(repository repository) *Service {
	return &Service{repository: repository}
}

// ReviewException updates a reconciliation exception state.
func (s *Service) ReviewException(ctx context.Context, params ReviewParams) error {
	params.ExceptionID = strings.TrimSpace(params.ExceptionID)
	params.Status = strings.TrimSpace(params.Status)
	params.ReviewerName = trimDefault(params.ReviewerName, "finance-reviewer")
	params.Note = strings.TrimSpace(params.Note)
	if params.ExceptionID == "" || !validReviewStatus(params.Status) {
		return fmt.Errorf("%w: invalid review state", ErrInvalidReview)
	}
	return s.repository.ReviewException(ctx, params)
}

// CreateCorrection stores a correction record for an exception.
func (s *Service) CreateCorrection(ctx context.Context, params CorrectionParams) (Correction, error) {
	params.ExceptionID = strings.TrimSpace(params.ExceptionID)
	params.Reason = strings.TrimSpace(params.Reason)
	params.ReviewerName = trimDefault(params.ReviewerName, "finance-reviewer")
	if params.ExceptionID == "" || params.Reason == "" {
		return Correction{}, fmt.Errorf("%w: missing correction target or reason", ErrInvalidReview)
	}
	return s.repository.CreateCorrection(ctx, params)
}

// ConfirmBill confirms a draft and resolves linked exceptions.
func (s *Service) ConfirmBill(ctx context.Context, params ConfirmParams) error {
	params.BillID = strings.TrimSpace(params.BillID)
	params.ReviewerName = trimDefault(params.ReviewerName, "finance-reviewer")
	params.Note = strings.TrimSpace(params.Note)
	if params.BillID == "" {
		return fmt.Errorf("%w: missing bill", ErrInvalidReview)
	}
	return s.repository.ConfirmBill(ctx, params)
}

// ExportReconciliation returns a reconciliations export package.
func (s *Service) ExportReconciliation(ctx context.Context, siteID string, generatedBy string) (ReconciliationExport, error) {
	generatedBy = trimDefault(generatedBy, "finance-reviewer")
	return s.repository.ExportReconciliation(ctx, strings.TrimSpace(siteID), generatedBy)
}

func validReviewStatus(status string) bool {
	switch status {
	case "open", "reviewing", "resolved":
		return true
	default:
		return false
	}
}

func trimDefault(value string, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	return trimmed
}
