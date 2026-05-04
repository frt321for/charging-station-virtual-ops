package billing

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"charging-ops/backend/internal/domain/auth"
)

type repository interface {
	ListPolicies(ctx context.Context, siteID string) ([]PricingPolicy, error)
	ListDrafts(ctx context.Context, siteID string) ([]BillingDraft, error)
	ListExceptions(ctx context.Context, siteID string) ([]ReconciliationException, error)
	FindSessionForDraft(ctx context.Context, sessionID string) (sessionForDraft, error)
	FindActivePolicy(ctx context.Context, siteID string, chargerType string) (PricingPolicy, error)
	SaveDraft(ctx context.Context, calculation draftCalculation, generatedBy string) (BillingDraft, error)
}

// Service coordinates pricing, billing draft, and reconciliation logic.
type Service struct {
	repository repository
}

// NewService creates a billing service.
func NewService(repository repository) *Service {
	return &Service{repository: repository}
}

// ListPolicies returns versioned pricing policies for a site.
func (s *Service) ListPolicies(ctx context.Context, siteID string) ([]PricingPolicy, error) {
	siteID = strings.TrimSpace(siteID)
	if !canQuerySite(ctx, siteID) {
		return nil, auth.ErrForbidden
	}
	policies, err := s.repository.ListPolicies(ctx, siteID)
	if err != nil {
		return nil, err
	}
	filtered := policies[:0]
	for _, policy := range policies {
		if auth.CanAccessSite(ctx, policy.SiteID, policy.SiteCode) {
			filtered = append(filtered, policy)
		}
	}
	return filtered, nil
}

// ListDrafts returns recent billing drafts.
func (s *Service) ListDrafts(ctx context.Context, siteID string) ([]BillingDraft, error) {
	siteID = strings.TrimSpace(siteID)
	if !canQuerySite(ctx, siteID) {
		return nil, auth.ErrForbidden
	}
	drafts, err := s.repository.ListDrafts(ctx, siteID)
	if err != nil {
		return nil, err
	}
	filtered := drafts[:0]
	for _, draft := range drafts {
		if auth.CanAccessSite(ctx, draft.SiteID, draft.SiteCode) {
			filtered = append(filtered, draft)
		}
	}
	return filtered, nil
}

// ListExceptions returns open reconciliation exceptions.
func (s *Service) ListExceptions(ctx context.Context, siteID string) ([]ReconciliationException, error) {
	siteID = strings.TrimSpace(siteID)
	if !canQuerySite(ctx, siteID) {
		return nil, auth.ErrForbidden
	}
	return s.repository.ListExceptions(ctx, siteID)
}

// GenerateDraft calculates and persists a billing draft for a session.
func (s *Service) GenerateDraft(ctx context.Context, params GenerateDraftParams) (BillingDraft, error) {
	sessionID := strings.TrimSpace(params.SessionID)
	if sessionID == "" {
		return BillingDraft{}, fmt.Errorf("%w: missing session id", ErrInvalidDraft)
	}

	session, err := s.repository.FindSessionForDraft(ctx, sessionID)
	if err != nil {
		return BillingDraft{}, err
	}
	if !auth.CanAccessSite(ctx, session.SiteID, session.SiteCode) {
		return BillingDraft{}, auth.ErrForbidden
	}
	if session.Status != "pending_billing" && session.Status != "pending_review" && session.Status != "billed" {
		return BillingDraft{}, fmt.Errorf("%w: session is not stopped", ErrInvalidDraft)
	}

	policy, err := s.repository.FindActivePolicy(ctx, session.SiteID, session.ChargerType)
	if err != nil {
		return BillingDraft{}, err
	}

	calculation := calculateDraft(session, policy)
	generatedBy := strings.TrimSpace(params.GeneratedBy)
	if generatedBy == "" {
		generatedBy = "billing-engine"
	}
	return s.repository.SaveDraft(ctx, calculation, generatedBy)
}

func canQuerySite(ctx context.Context, siteID string) bool {
	principal, ok := auth.PrincipalFromContext(ctx)
	if !ok || principal.HasAnySiteAccess() {
		return true
	}
	return siteID != "" && principal.CanAccessSite(siteID, siteID)
}

func calculateDraft(session sessionForDraft, policy PricingPolicy) draftCalculation {
	energy := calculateEnergy(session)
	duration := calculateDuration(session)
	period := selectPricingPeriod(policy, session.StoppedAt)

	energyAmount := roundMoney(energy * period.EnergyPricePerKWh)
	serviceAmount := roundMoney(energy * period.ServiceFeePerKWh)
	occupancyAmount := roundMoney(float64(duration) * period.OccupancyFeePerMinute)
	total := roundMoney(energyAmount + serviceAmount + occupancyAmount)

	exceptions := detectExceptions(session, energy, duration, total)
	return draftCalculation{
		Session:         session,
		Policy:          policy,
		EnergyKWh:       round3(energy),
		DurationMinutes: duration,
		EnergyAmount:    energyAmount,
		ServiceAmount:   serviceAmount,
		OccupancyAmount: occupancyAmount,
		TotalAmount:     total,
		ExceptionFlag:   len(exceptions) > 0,
		Exceptions:      exceptions,
	}
}

func calculateEnergy(session sessionForDraft) float64 {
	start := firstPresent(session.MeterStartKWh, session.FirstMeterKWh)
	stop := firstPresent(session.MeterStopKWh, session.LastMeterKWh)
	if start == nil || stop == nil || *stop < *start {
		return 0
	}
	return *stop - *start
}

func calculateDuration(session sessionForDraft) int {
	if session.StartedAt == nil || session.StoppedAt == nil {
		return 0
	}
	minutes := int(math.Ceil(session.StoppedAt.Sub(*session.StartedAt).Minutes()))
	if minutes < 0 {
		return 0
	}
	return minutes
}

func selectPricingPeriod(policy PricingPolicy, stoppedAt *time.Time) PricingPeriod {
	if len(policy.Periods) == 0 {
		return PricingPeriod{}
	}
	if stoppedAt == nil {
		return policy.Periods[0]
	}
	minute := stoppedAt.Hour()*60 + stoppedAt.Minute()
	for _, period := range policy.Periods {
		if minute >= period.StartMinute && minute < period.EndMinute {
			return period
		}
	}
	return policy.Periods[0]
}

func detectExceptions(session sessionForDraft, energy float64, duration int, total float64) []exceptionDraft {
	exceptions := make([]exceptionDraft, 0)
	if energy <= 0 {
		exceptions = append(exceptions, exceptionDraft{
			Type:            "energy",
			Severity:        "high",
			Reason:          "电量差值为 0 或无法从读数计算",
			SuggestedAction: "核对电表起止读数与采样上报",
		})
	}
	if duration <= 0 {
		exceptions = append(exceptions, exceptionDraft{
			Type:            "duration",
			Severity:        "medium",
			Reason:          "会话缺少有效起止时间",
			SuggestedAction: "核对启动与停止事件时间线",
		})
	}
	if total <= 0 {
		exceptions = append(exceptions, exceptionDraft{
			Type:            "amount",
			Severity:        "medium",
			Reason:          "账单金额为 0",
			SuggestedAction: "核对定价策略和计费周期",
		})
	}
	if strings.TrimSpace(session.StopReason) == "" {
		exceptions = append(exceptions, exceptionDraft{
			Type:            "stop_reason",
			Severity:        "low",
			Reason:          "停止原因为空",
			SuggestedAction: "补充远程停止或设备停止原因",
		})
	}
	return exceptions
}

func firstPresent(values ...*float64) *float64 {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func roundMoney(value float64) float64 {
	return math.Round(value*100) / 100
}

func round3(value float64) float64 {
	return math.Round(value*1000) / 1000
}
