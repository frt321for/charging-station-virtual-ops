package session

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"charging-ops/backend/internal/domain/auth"
)

type repository interface {
	List(ctx context.Context) ([]Summary, error)
	Get(ctx context.Context, sessionID string) (Detail, error)
	GetBoundary(ctx context.Context, sessionID string) (siteBoundary, error)
	GetStatus(ctx context.Context, sessionID string) (Status, error)
	GetConnectorBoundary(ctx context.Context, connectorCode string) (siteBoundary, error)
	CreateReservation(ctx context.Context, params CreateReservationParams) (string, error)
	AppendTransition(ctx context.Context, params TransitionParams) error
}

type siteBoundary struct {
	SiteID   string
	SiteCode string
}

// Service coordinates charging session lifecycle operations.
type Service struct {
	repository repository
}

// NewService creates a charging session service.
func NewService(repository repository) *Service {
	return &Service{repository: repository}
}

// List returns recent charging sessions.
func (s *Service) List(ctx context.Context) ([]Summary, error) {
	sessions, err := s.repository.List(ctx)
	if err != nil {
		return nil, err
	}
	filtered := sessions[:0]
	for _, summary := range sessions {
		if auth.CanAccessSite(ctx, summary.SiteID, "") {
			filtered = append(filtered, summary)
		}
	}
	return filtered, nil
}

// Get returns one charging session with its timeline.
func (s *Service) Get(ctx context.Context, sessionID string) (Detail, error) {
	detail, err := s.repository.Get(ctx, sessionID)
	if err != nil {
		return Detail{}, err
	}
	if !auth.CanAccessSite(ctx, detail.SiteID, "") {
		return Detail{}, auth.ErrForbidden
	}
	return detail, nil
}

// Transition validates and persists a lifecycle transition.
func (s *Service) Transition(ctx context.Context, params TransitionParams) (Detail, error) {
	boundary, err := s.repository.GetBoundary(ctx, params.SessionID)
	if err != nil {
		return Detail{}, err
	}
	if !auth.CanAccessSite(ctx, boundary.SiteID, boundary.SiteCode) {
		return Detail{}, auth.ErrForbidden
	}

	current, err := s.repository.GetStatus(ctx, params.SessionID)
	if err != nil {
		return Detail{}, err
	}

	if err := ValidateTransition(current, params.TargetStatus); err != nil {
		return Detail{}, err
	}

	normalized := normalizeTransitionParams(params)
	if err := s.repository.AppendTransition(ctx, normalized); err != nil {
		return Detail{}, err
	}

	return s.Get(ctx, params.SessionID)
}

// CreateReservation creates a waiting-arrival session for a connector.
func (s *Service) CreateReservation(ctx context.Context, params CreateReservationParams) (Detail, error) {
	params = normalizeReservationParams(params)
	if params.ConnectorCode == "" {
		return Detail{}, fmt.Errorf("%w: missing connector code", ErrInvalidReservation)
	}
	boundary, err := s.repository.GetConnectorBoundary(ctx, params.ConnectorCode)
	if err != nil {
		return Detail{}, err
	}
	if !auth.CanAccessSite(ctx, boundary.SiteID, boundary.SiteCode) {
		return Detail{}, auth.ErrForbidden
	}

	sessionNo, err := s.repository.CreateReservation(ctx, params)
	if err != nil {
		return Detail{}, err
	}
	return s.Get(ctx, sessionNo)
}

// ErrInvalidReservation is returned when a reservation request is invalid.
var ErrInvalidReservation = errors.New("invalid reservation")

func normalizeReservationParams(params CreateReservationParams) CreateReservationParams {
	params.ConnectorCode = strings.TrimSpace(params.ConnectorCode)
	params.RequestedBy = strings.TrimSpace(params.RequestedBy)
	if params.RequestedBy == "" {
		params.RequestedBy = "operations"
	}
	if params.ReservationMinutes <= 0 {
		params.ReservationMinutes = 30
	}
	if len(params.Payload) == 0 {
		params.Payload = json.RawMessage(`{}`)
	}
	return params
}

func normalizeTransitionParams(params TransitionParams) TransitionParams {
	if params.EventType == "" {
		params.EventType = "StatusTransitioned"
	}
	if params.Source == "" {
		params.Source = "operations"
	}
	if len(params.Payload) == 0 {
		params.Payload = json.RawMessage(`{}`)
	}
	return params
}
