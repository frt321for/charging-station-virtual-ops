package billing

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestCalculateDraft(t *testing.T) {
	startedAt := time.Date(2026, 5, 4, 8, 0, 0, 0, time.UTC)
	stoppedAt := startedAt.Add(74 * time.Minute)
	calculation := calculateDraft(sessionForDraft{
		SessionID:     "session-1",
		SessionNo:     "CS-1",
		SiteID:        "site-1",
		SiteCode:      "HQ-CAMPUS",
		ConnectorCode: "AC-N-001-01",
		ChargerType:   "ac",
		Status:        "pending_billing",
		StartedAt:     &startedAt,
		StoppedAt:     &stoppedAt,
		StopReason:    "remote_stop",
		MeterStartKWh: floatPtr(100.25),
		MeterStopKWh:  floatPtr(113.75),
	}, PricingPolicy{
		ID:      "policy-1",
		Version: 1,
		Periods: []PricingPeriod{
			{
				StartMinute:           0,
				EndMinute:             1440,
				EnergyPricePerKWh:     1.12,
				ServiceFeePerKWh:      0.28,
				OccupancyFeePerMinute: 0.03,
			},
		},
	})

	if calculation.EnergyKWh != 13.5 {
		t.Fatalf("expected energy 13.5, got %.3f", calculation.EnergyKWh)
	}
	if calculation.DurationMinutes != 74 {
		t.Fatalf("expected duration 74, got %d", calculation.DurationMinutes)
	}
	if calculation.TotalAmount != 21.12 {
		t.Fatalf("expected total 21.12, got %.2f", calculation.TotalAmount)
	}
	if calculation.ExceptionFlag {
		t.Fatalf("expected no exception")
	}
}

func TestCalculateDraftDetectsExceptions(t *testing.T) {
	calculation := calculateDraft(sessionForDraft{
		SessionID: "session-1",
		Status:    "pending_billing",
	}, PricingPolicy{Periods: []PricingPeriod{{StartMinute: 0, EndMinute: 1440}}})

	if !calculation.ExceptionFlag {
		t.Fatalf("expected exception flag")
	}
	if len(calculation.Exceptions) != 4 {
		t.Fatalf("expected 4 exceptions, got %d", len(calculation.Exceptions))
	}
}

func TestServiceGenerateDraft(t *testing.T) {
	tests := []struct {
		name        string
		session     sessionForDraft
		wantErr     error
		wantSaved   bool
		generatedBy string
	}{
		{
			name:      "saves pending billing draft",
			session:   sessionForDraft{SessionID: "session-1", SiteID: "site-1", ChargerType: "ac", Status: "pending_billing"},
			wantSaved: true,
		},
		{
			name:    "rejects active session",
			session: sessionForDraft{SessionID: "session-1", SiteID: "site-1", ChargerType: "ac", Status: "charging"},
			wantErr: ErrInvalidDraft,
		},
		{
			name:      "defaults generator",
			session:   sessionForDraft{SessionID: "session-1", SiteID: "site-1", ChargerType: "ac", Status: "pending_review"},
			wantSaved: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := &fakeRepository{session: test.session}
			service := NewService(repository)

			_, err := service.GenerateDraft(context.Background(), GenerateDraftParams{
				SessionID:   "session-1",
				GeneratedBy: test.generatedBy,
			})

			if test.wantErr == nil && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if test.wantErr != nil && !errors.Is(err, test.wantErr) {
				t.Fatalf("expected %v, got %v", test.wantErr, err)
			}
			if repository.saved != test.wantSaved {
				t.Fatalf("expected saved=%v, got %v", test.wantSaved, repository.saved)
			}
			if test.wantSaved && repository.generatedBy != "billing-engine" {
				t.Fatalf("expected billing-engine generator, got %s", repository.generatedBy)
			}
		})
	}
}

type fakeRepository struct {
	session     sessionForDraft
	saved       bool
	generatedBy string
}

func (f *fakeRepository) ListPolicies(ctx context.Context, siteID string) ([]PricingPolicy, error) {
	return []PricingPolicy{}, nil
}

func (f *fakeRepository) ListDrafts(ctx context.Context, siteID string) ([]BillingDraft, error) {
	return []BillingDraft{}, nil
}

func (f *fakeRepository) ListExceptions(ctx context.Context, siteID string) ([]ReconciliationException, error) {
	return []ReconciliationException{}, nil
}

func (f *fakeRepository) FindSessionForDraft(ctx context.Context, sessionID string) (sessionForDraft, error) {
	return f.session, nil
}

func (f *fakeRepository) FindActivePolicy(ctx context.Context, siteID string, chargerType string) (PricingPolicy, error) {
	return PricingPolicy{
		ID:      "policy-1",
		Periods: []PricingPeriod{{StartMinute: 0, EndMinute: 1440}},
	}, nil
}

func (f *fakeRepository) SaveDraft(
	ctx context.Context,
	calculation draftCalculation,
	generatedBy string,
) (BillingDraft, error) {
	f.saved = true
	f.generatedBy = generatedBy
	return BillingDraft{SessionID: calculation.Session.SessionID}, nil
}

func floatPtr(value float64) *float64 {
	return &value
}
