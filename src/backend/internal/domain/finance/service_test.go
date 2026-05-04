package finance

import (
	"context"
	"errors"
	"testing"
	"time"

	"charging-ops/backend/internal/domain/auth"
)

func TestValidReviewStatus(t *testing.T) {
	if !validReviewStatus("reviewing") {
		t.Fatalf("expected reviewing to be valid")
	}
	if validReviewStatus("confirmed") {
		t.Fatalf("expected confirmed to be invalid")
	}
}

func TestBuildExport(t *testing.T) {
	export := buildExport([]ExportRow{{
		ExceptionNo:   "RE-1",
		BillNo:        "BILL-1",
		SessionNo:     "CS-1",
		ExceptionType: "amount",
		Severity:      "medium",
		Status:        "open",
		Reason:        "amount mismatch",
		TotalAmount:   12.5,
	}}, "finance")

	if export.ExportNo == "" || export.CSV == "" || len(export.Rows) != 1 {
		t.Fatalf("unexpected export: %#v", export)
	}
}

func TestServiceReviewExceptionRejectsUnauthorizedBeforeWrite(t *testing.T) {
	repository := &fakeFinanceRepository{exceptionBoundary: siteBoundary{SiteID: "site-2", SiteCode: "OTHER"}}
	service := NewService(repository)

	err := service.ReviewException(siteLimitedContext(), ReviewParams{
		ExceptionID:  "RE-1",
		Status:       "reviewing",
		ReviewerName: "finance",
	})

	if !errors.Is(err, auth.ErrForbidden) {
		t.Fatalf("expected forbidden, got %v", err)
	}
	if repository.reviewed {
		t.Fatalf("expected no review write before authorization")
	}
}

func TestServiceCreateCorrectionRejectsUnauthorizedBeforeWrite(t *testing.T) {
	repository := &fakeFinanceRepository{exceptionBoundary: siteBoundary{SiteID: "site-2", SiteCode: "OTHER"}}
	service := NewService(repository)

	_, err := service.CreateCorrection(siteLimitedContext(), CorrectionParams{
		ExceptionID:  "RE-1",
		Reason:       "amount mismatch",
		ReviewerName: "finance",
	})

	if !errors.Is(err, auth.ErrForbidden) {
		t.Fatalf("expected forbidden, got %v", err)
	}
	if repository.corrected {
		t.Fatalf("expected no correction write before authorization")
	}
}

func TestServiceConfirmBillRejectsUnauthorizedBeforeWrite(t *testing.T) {
	repository := &fakeFinanceRepository{billBoundary: siteBoundary{SiteID: "site-2", SiteCode: "OTHER"}}
	service := NewService(repository)

	err := service.ConfirmBill(siteLimitedContext(), ConfirmParams{
		BillID:       "BILL-1",
		ReviewerName: "finance",
	})

	if !errors.Is(err, auth.ErrForbidden) {
		t.Fatalf("expected forbidden, got %v", err)
	}
	if repository.confirmed {
		t.Fatalf("expected no bill confirm write before authorization")
	}
}

type fakeFinanceRepository struct {
	exceptionBoundary siteBoundary
	billBoundary      siteBoundary
	reviewed          bool
	corrected         bool
	confirmed         bool
}

func (f *fakeFinanceRepository) FindExceptionBoundary(ctx context.Context, exceptionID string) (siteBoundary, error) {
	if f.exceptionBoundary.SiteID == "" {
		return siteBoundary{SiteID: "site-1", SiteCode: "HQ-CAMPUS"}, nil
	}
	return f.exceptionBoundary, nil
}

func (f *fakeFinanceRepository) FindBillBoundary(ctx context.Context, billID string) (siteBoundary, error) {
	if f.billBoundary.SiteID == "" {
		return siteBoundary{SiteID: "site-1", SiteCode: "HQ-CAMPUS"}, nil
	}
	return f.billBoundary, nil
}

func (f *fakeFinanceRepository) ReviewException(ctx context.Context, params ReviewParams) error {
	f.reviewed = true
	return nil
}

func (f *fakeFinanceRepository) CreateCorrection(ctx context.Context, params CorrectionParams) (Correction, error) {
	f.corrected = true
	return Correction{ID: "correction-1", CreatedAt: time.Now().UTC()}, nil
}

func (f *fakeFinanceRepository) ConfirmBill(ctx context.Context, params ConfirmParams) error {
	f.confirmed = true
	return nil
}

func (f *fakeFinanceRepository) ExportReconciliation(ctx context.Context, siteID string, generatedBy string) (ReconciliationExport, error) {
	return ReconciliationExport{}, nil
}

func siteLimitedContext() context.Context {
	return auth.WithPrincipal(context.Background(), auth.Principal{
		Permissions: []string{auth.PermissionFinanceReview},
		Sites:       []auth.AuthorizedSite{{ID: "site-1", Code: "HQ-CAMPUS"}},
	})
}
