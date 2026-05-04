package maintenance

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"charging-ops/backend/internal/domain/auth"
)

type repository interface {
	Snapshot(ctx context.Context, siteID string) (Snapshot, error)
	FindFaultBoundary(ctx context.Context, faultID string) (siteBoundary, error)
	FindWorkOrderBoundary(ctx context.Context, workOrderID string) (siteBoundary, error)
	ListEvents(ctx context.Context, workOrderID string) ([]WorkOrderEvent, error)
	CreateWorkOrder(ctx context.Context, params CreateWorkOrderParams) (WorkOrder, error)
	TransitionWorkOrder(ctx context.Context, params TransitionParams) (WorkOrder, error)
}

type siteBoundary struct {
	SiteID   string
	SiteCode string
}

// Service coordinates fault handling, work orders, and SLA policy checks.
type Service struct {
	repository repository
}

// NewService creates a maintenance service.
func NewService(repository repository) *Service {
	return &Service{repository: repository}
}

// Snapshot returns the maintenance workbench state for a site.
func (s *Service) Snapshot(ctx context.Context, siteID string) (Snapshot, error) {
	snapshot, err := s.repository.Snapshot(ctx, strings.TrimSpace(siteID))
	if err != nil {
		return Snapshot{}, err
	}
	if !auth.CanAccessSite(ctx, snapshot.SLA.SiteID, snapshot.SLA.SiteCode) {
		return Snapshot{}, auth.ErrForbidden
	}
	return snapshot, nil
}

// ListEvents returns a work-order timeline.
func (s *Service) ListEvents(ctx context.Context, workOrderID string) ([]WorkOrderEvent, error) {
	workOrderID = strings.TrimSpace(workOrderID)
	if workOrderID == "" {
		return nil, fmt.Errorf("%w: missing work order", ErrInvalidWorkOrder)
	}
	if err := s.authorizeWorkOrder(ctx, workOrderID); err != nil {
		return nil, err
	}
	return s.repository.ListEvents(ctx, workOrderID)
}

// CreateWorkOrder creates or returns the linked work order for a fault.
func (s *Service) CreateWorkOrder(ctx context.Context, params CreateWorkOrderParams) (WorkOrder, error) {
	params.FaultID = strings.TrimSpace(params.FaultID)
	params.AssigneeName = strings.TrimSpace(params.AssigneeName)
	params.ImpactScope = trimDefault(params.ImpactScope, "charger")
	params.Title = strings.TrimSpace(params.Title)
	params.Description = strings.TrimSpace(params.Description)
	params.ActorName = trimDefault(params.ActorName, "maintenance")
	if params.FaultID == "" {
		return WorkOrder{}, fmt.Errorf("%w: missing fault", ErrInvalidWorkOrder)
	}
	if err := s.authorizeFault(ctx, params.FaultID); err != nil {
		return WorkOrder{}, err
	}
	workOrder, err := s.repository.CreateWorkOrder(ctx, params)
	if err != nil {
		return WorkOrder{}, err
	}
	if !auth.CanAccessSite(ctx, workOrder.SiteID, workOrder.SiteCode) {
		return WorkOrder{}, auth.ErrForbidden
	}
	return workOrder, nil
}

// TransitionWorkOrder advances a maintenance ticket lifecycle.
func (s *Service) TransitionWorkOrder(ctx context.Context, params TransitionParams) (WorkOrder, error) {
	params.WorkOrderID = strings.TrimSpace(params.WorkOrderID)
	params.TargetStatus = strings.TrimSpace(params.TargetStatus)
	params.AssigneeName = strings.TrimSpace(params.AssigneeName)
	params.ActorName = trimDefault(params.ActorName, "maintenance")
	params.Note = strings.TrimSpace(params.Note)
	if len(params.Payload) == 0 {
		params.Payload = json.RawMessage(`{}`)
	}
	if params.WorkOrderID == "" || !validStatus(params.TargetStatus) {
		return WorkOrder{}, fmt.Errorf("%w: invalid transition", ErrInvalidWorkOrder)
	}
	if err := s.authorizeWorkOrder(ctx, params.WorkOrderID); err != nil {
		return WorkOrder{}, err
	}
	workOrder, err := s.repository.TransitionWorkOrder(ctx, params)
	if err != nil {
		return WorkOrder{}, err
	}
	if !auth.CanAccessSite(ctx, workOrder.SiteID, workOrder.SiteCode) {
		return WorkOrder{}, auth.ErrForbidden
	}
	return workOrder, nil
}

func (s *Service) authorizeFault(ctx context.Context, faultID string) error {
	boundary, err := s.repository.FindFaultBoundary(ctx, faultID)
	if err != nil {
		return err
	}
	if !auth.CanAccessSite(ctx, boundary.SiteID, boundary.SiteCode) {
		return auth.ErrForbidden
	}
	return nil
}

func (s *Service) authorizeWorkOrder(ctx context.Context, workOrderID string) error {
	boundary, err := s.repository.FindWorkOrderBoundary(ctx, workOrderID)
	if err != nil {
		return err
	}
	if !auth.CanAccessSite(ctx, boundary.SiteID, boundary.SiteCode) {
		return auth.ErrForbidden
	}
	return nil
}

func validStatus(status string) bool {
	switch status {
	case "open", "assigned", "accepted", "arrived", "handling", "retest", "recovered", "closed", "cancelled":
		return true
	default:
		return false
	}
}

func validTransition(current string, target string) bool {
	if current == target {
		return true
	}
	switch current {
	case "open":
		return target == "assigned" || target == "cancelled"
	case "assigned":
		return target == "accepted" || target == "cancelled"
	case "accepted":
		return target == "arrived" || target == "cancelled"
	case "arrived":
		return target == "handling" || target == "cancelled"
	case "handling":
		return target == "retest" || target == "cancelled"
	case "retest":
		return target == "recovered" || target == "handling" || target == "cancelled"
	case "recovered":
		return target == "closed"
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
