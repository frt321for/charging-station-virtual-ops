package finance

import (
	"context"
	"fmt"
	"strings"

	"charging-ops/backend/internal/domain/auth"
)

type repository interface {
	FindExceptionBoundary(ctx context.Context, exceptionID string) (siteBoundary, error)
	FindBillBoundary(ctx context.Context, billID string) (siteBoundary, error)
	ReviewException(ctx context.Context, params ReviewParams) error
	CreateCorrection(ctx context.Context, params CorrectionParams) (Correction, error)
	ConfirmBill(ctx context.Context, params ConfirmParams) error
	ExportReconciliation(ctx context.Context, siteID string, generatedBy string) (ReconciliationExport, error)
}

type siteBoundary struct {
	SiteID   string
	SiteCode string
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
	if err := s.authorizeException(ctx, params.ExceptionID); err != nil {
		return err
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
	if err := s.authorizeException(ctx, params.ExceptionID); err != nil {
		return Correction{}, err
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
	if err := s.authorizeBill(ctx, params.BillID); err != nil {
		return err
	}
	return s.repository.ConfirmBill(ctx, params)
}

// ExportReconciliation returns a reconciliations export package.
func (s *Service) ExportReconciliation(ctx context.Context, siteID string, generatedBy string) (ReconciliationExport, error) {
	siteID = strings.TrimSpace(siteID)
	if !canQuerySite(ctx, siteID) {
		return ReconciliationExport{}, auth.ErrForbidden
	}
	generatedBy = trimDefault(generatedBy, "finance-reviewer")
	return s.repository.ExportReconciliation(ctx, siteID, generatedBy)
}

func canQuerySite(ctx context.Context, siteID string) bool {
	principal, ok := auth.PrincipalFromContext(ctx)
	if !ok || principal.HasAnySiteAccess() {
		return true
	}
	return siteID != "" && principal.CanAccessSite(siteID, siteID)
}

func (s *Service) authorizeException(ctx context.Context, exceptionID string) error {
	boundary, err := s.repository.FindExceptionBoundary(ctx, exceptionID)
	if err != nil {
		return err
	}
	if !auth.CanAccessSite(ctx, boundary.SiteID, boundary.SiteCode) {
		return auth.ErrForbidden
	}
	return nil
}

func (s *Service) authorizeBill(ctx context.Context, billID string) error {
	boundary, err := s.repository.FindBillBoundary(ctx, billID)
	if err != nil {
		return err
	}
	if !auth.CanAccessSite(ctx, boundary.SiteID, boundary.SiteCode) {
		return auth.ErrForbidden
	}
	return nil
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
