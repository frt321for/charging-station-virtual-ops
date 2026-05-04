package configmgmt

import (
	"context"
	"strings"

	"charging-ops/backend/internal/domain/auth"
)

type repository interface {
	FindSiteBoundary(ctx context.Context, siteID string) (Boundary, error)
	FindBoundary(ctx context.Context, entity string, id string) (Boundary, error)
	Snapshot(ctx context.Context, siteID string) (Snapshot, error)
	UpdateSite(ctx context.Context, siteID string, params SiteUpdate) (SiteConfig, error)
	UpdateArea(ctx context.Context, id string, params AreaUpdate) (AreaConfig, error)
	UpdateGroup(ctx context.Context, id string, params GroupUpdate) (GroupConfig, error)
	UpdateCharger(ctx context.Context, id string, params ChargerUpdate) (ChargerConfig, error)
	UpdateConnector(ctx context.Context, id string, params ConnectorUpdate) (ConnectorConfig, error)
	UpdateLoadPolicy(ctx context.Context, id string, params LoadPolicyUpdate) (LoadPolicy, error)
	UpdateReservationRule(ctx context.Context, id string, params ReservationRuleUpdate) (ReservationRule, error)
	UpdateQueueRule(ctx context.Context, id string, params QueueRuleUpdate) (QueueRule, error)
	CreatePricingPolicy(ctx context.Context, params PricingPolicyCreate) (PricingPolicy, error)
}

// Service enforces object boundaries before applying configuration changes.
type Service struct {
	repository repository
}

// NewService creates a configuration service.
func NewService(repository repository) *Service {
	return &Service{repository: repository}
}

// Snapshot returns editable configuration for one authorized site.
func (s *Service) Snapshot(ctx context.Context, siteID string) (Snapshot, error) {
	boundary, err := s.repository.FindSiteBoundary(ctx, siteID)
	if err != nil {
		return Snapshot{}, err
	}
	if !auth.CanAccessSite(ctx, boundary.SiteID, boundary.SiteCode) {
		return Snapshot{}, ErrForbidden
	}
	return s.repository.Snapshot(ctx, boundary.SiteID)
}

func (s *Service) UpdateSite(ctx context.Context, siteID string, params SiteUpdate) (SiteConfig, error) {
	if err := validateSiteUpdate(params); err != nil {
		return SiteConfig{}, err
	}
	boundary, err := s.repository.FindSiteBoundary(ctx, siteID)
	if err != nil {
		return SiteConfig{}, err
	}
	if !auth.CanAccessSite(ctx, boundary.SiteID, boundary.SiteCode) {
		return SiteConfig{}, ErrForbidden
	}
	return s.repository.UpdateSite(ctx, boundary.SiteID, params)
}

func (s *Service) UpdateArea(ctx context.Context, id string, params AreaUpdate) (AreaConfig, error) {
	if err := validatePositiveFloat(params.LoadLimitKW); err != nil {
		return AreaConfig{}, err
	}
	if err := s.ensureBoundary(ctx, "area", id); err != nil {
		return AreaConfig{}, err
	}
	return s.repository.UpdateArea(ctx, id, params)
}

func (s *Service) UpdateGroup(ctx context.Context, id string, params GroupUpdate) (GroupConfig, error) {
	if err := validatePositiveFloat(params.LoadLimitKW); err != nil {
		return GroupConfig{}, err
	}
	if err := s.ensureBoundary(ctx, "group", id); err != nil {
		return GroupConfig{}, err
	}
	return s.repository.UpdateGroup(ctx, id, params)
}

func (s *Service) UpdateCharger(ctx context.Context, id string, params ChargerUpdate) (ChargerConfig, error) {
	if err := validatePositiveFloat(params.RatedPowerKW); err != nil {
		return ChargerConfig{}, err
	}
	if err := s.ensureBoundary(ctx, "charger", id); err != nil {
		return ChargerConfig{}, err
	}
	return s.repository.UpdateCharger(ctx, id, params)
}

func (s *Service) UpdateConnector(ctx context.Context, id string, params ConnectorUpdate) (ConnectorConfig, error) {
	if err := validatePositiveFloat(params.MaxPowerKW); err != nil {
		return ConnectorConfig{}, err
	}
	if err := s.ensureBoundary(ctx, "connector", id); err != nil {
		return ConnectorConfig{}, err
	}
	return s.repository.UpdateConnector(ctx, id, params)
}

func (s *Service) UpdateLoadPolicy(ctx context.Context, id string, params LoadPolicyUpdate) (LoadPolicy, error) {
	if err := validatePositiveFloat(params.ThresholdKW); err != nil {
		return LoadPolicy{}, err
	}
	if err := validatePositiveFloat(params.WarningKW); err != nil {
		return LoadPolicy{}, err
	}
	if err := s.ensureBoundary(ctx, "load_policy", id); err != nil {
		return LoadPolicy{}, err
	}
	return s.repository.UpdateLoadPolicy(ctx, id, params)
}

func (s *Service) UpdateReservationRule(ctx context.Context, id string, params ReservationRuleUpdate) (ReservationRule, error) {
	if err := validatePositiveInt(params.HoldMinutes); err != nil {
		return ReservationRule{}, err
	}
	if err := s.ensureBoundary(ctx, "reservation_rule", id); err != nil {
		return ReservationRule{}, err
	}
	return s.repository.UpdateReservationRule(ctx, id, params)
}

func (s *Service) UpdateQueueRule(ctx context.Context, id string, params QueueRuleUpdate) (QueueRule, error) {
	if err := validatePositiveInt(params.MaxQueueSize); err != nil {
		return QueueRule{}, err
	}
	if err := s.ensureBoundary(ctx, "queue_rule", id); err != nil {
		return QueueRule{}, err
	}
	return s.repository.UpdateQueueRule(ctx, id, params)
}

func (s *Service) CreatePricingPolicy(ctx context.Context, params PricingPolicyCreate) (PricingPolicy, error) {
	params.Code = strings.TrimSpace(params.Code)
	params.Name = strings.TrimSpace(params.Name)
	params.SiteID = strings.TrimSpace(params.SiteID)
	params.ChargerType = trimDefault(params.ChargerType, "all")
	params.Status = trimDefault(params.Status, "active")
	if params.SiteID == "" || params.Code == "" || params.Name == "" || len(params.Periods) == 0 {
		return PricingPolicy{}, ErrInvalidRequest
	}
	if !validIn(params.ChargerType, "all", "ac", "dc") || !validIn(params.Status, "draft", "active", "retired") {
		return PricingPolicy{}, ErrInvalidRequest
	}
	for _, period := range params.Periods {
		if period.Label == "" || period.StartMinute < 0 || period.EndMinute > 1440 || period.EndMinute <= period.StartMinute {
			return PricingPolicy{}, ErrInvalidRequest
		}
		if period.EnergyPricePerKWh < 0 || period.ServiceFeePerKWh < 0 || period.OccupancyFeePerMinute < 0 {
			return PricingPolicy{}, ErrInvalidRequest
		}
	}
	boundary, err := s.repository.FindSiteBoundary(ctx, params.SiteID)
	if err != nil {
		return PricingPolicy{}, err
	}
	if !auth.CanAccessSite(ctx, boundary.SiteID, boundary.SiteCode) {
		return PricingPolicy{}, ErrForbidden
	}
	params.SiteID = boundary.SiteID
	return s.repository.CreatePricingPolicy(ctx, params)
}

func (s *Service) ensureBoundary(ctx context.Context, entity string, id string) error {
	boundary, err := s.repository.FindBoundary(ctx, entity, id)
	if err != nil {
		return err
	}
	if !auth.CanAccessSite(ctx, boundary.SiteID, boundary.SiteCode) {
		return ErrForbidden
	}
	return nil
}

func validateSiteUpdate(params SiteUpdate) error {
	for _, value := range []*float64{params.CapacityKW, params.LoadLimitKW} {
		if err := validatePositiveFloat(value); err != nil {
			return err
		}
	}
	for _, value := range []*int{params.SLAResponseMinutes, params.SLARecoveryMinutes} {
		if err := validatePositiveInt(value); err != nil {
			return err
		}
	}
	return nil
}

func validatePositiveFloat(value *float64) error {
	if value != nil && *value <= 0 {
		return ErrInvalidRequest
	}
	return nil
}

func validatePositiveInt(value *int) error {
	if value != nil && *value <= 0 {
		return ErrInvalidRequest
	}
	return nil
}

func trimDefault(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func validIn(value string, allowed ...string) bool {
	for _, item := range allowed {
		if value == item {
			return true
		}
	}
	return false
}
