package loadcontrol

import (
	"context"
	"fmt"
	"strings"
)

type repository interface {
	Snapshot(ctx context.Context, siteID string) (Snapshot, error)
	CreateRecord(ctx context.Context, params CreateRecordParams) (LoadControlRecord, error)
}

// Service coordinates load-policy, queue, and control-record behavior.
type Service struct {
	repository repository
}

// NewService creates a load-control service.
func NewService(repository repository) *Service {
	return &Service{repository: repository}
}

// Snapshot returns the load-control console state for a site.
func (s *Service) Snapshot(ctx context.Context, siteID string) (Snapshot, error) {
	return s.repository.Snapshot(ctx, strings.TrimSpace(siteID))
}

// CreateRecord validates and persists a manual load-control decision.
func (s *Service) CreateRecord(ctx context.Context, params CreateRecordParams) (LoadControlRecord, error) {
	params.SiteID = strings.TrimSpace(params.SiteID)
	params.SessionID = strings.TrimSpace(params.SessionID)
	params.ActionType = strings.TrimSpace(params.ActionType)
	params.Reason = strings.TrimSpace(params.Reason)
	params.OperatorName = strings.TrimSpace(params.OperatorName)
	params.Status = strings.TrimSpace(params.Status)

	if params.SiteID == "" {
		return LoadControlRecord{}, fmt.Errorf("%w: missing site id", ErrInvalidRecord)
	}
	if !validAction(params.ActionType) {
		return LoadControlRecord{}, fmt.Errorf("%w: invalid action", ErrInvalidRecord)
	}
	if params.Reason == "" {
		params.Reason = "manual-load-control"
	}
	if params.OperatorName == "" {
		params.OperatorName = "operations"
	}
	if params.Status == "" {
		params.Status = "applied"
	}
	if !validStatus(params.Status) {
		return LoadControlRecord{}, fmt.Errorf("%w: invalid status", ErrInvalidRecord)
	}
	if params.BeforeLoadKW < 0 || params.AfterLoadKW < 0 {
		return LoadControlRecord{}, fmt.Errorf("%w: invalid load value", ErrInvalidRecord)
	}
	return s.repository.CreateRecord(ctx, params)
}

func validAction(action string) bool {
	switch action {
	case "limit_power", "pause", "resume", "queue", "reject", "promote", "release":
		return true
	default:
		return false
	}
}

func validStatus(status string) bool {
	switch status {
	case "recommended", "sent", "applied", "rejected":
		return true
	default:
		return false
	}
}
