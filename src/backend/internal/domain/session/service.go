package session

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

type repository interface {
	List(ctx context.Context) ([]Summary, error)
	Get(ctx context.Context, sessionID string) (Detail, error)
	GetStatus(ctx context.Context, sessionID string) (Status, error)
	CreateReservation(ctx context.Context, params CreateReservationParams) (string, error)
	AppendTransition(ctx context.Context, params TransitionParams) error
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
	return s.repository.List(ctx)
}

// Get returns one charging session with its timeline.
func (s *Service) Get(ctx context.Context, sessionID string) (Detail, error) {
	return s.repository.Get(ctx, sessionID)
}

// Transition validates and persists a lifecycle transition.
func (s *Service) Transition(ctx context.Context, params TransitionParams) (Detail, error) {
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

	return s.repository.Get(ctx, params.SessionID)
}

// CreateReservation creates a waiting-arrival session for a connector.
func (s *Service) CreateReservation(ctx context.Context, params CreateReservationParams) (Detail, error) {
	params = normalizeReservationParams(params)
	if params.ConnectorCode == "" {
		return Detail{}, fmt.Errorf("%w: missing connector code", ErrInvalidReservation)
	}

	sessionNo, err := s.repository.CreateReservation(ctx, params)
	if err != nil {
		return Detail{}, err
	}
	return s.repository.Get(ctx, sessionNo)
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
