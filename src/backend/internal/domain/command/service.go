package command

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"charging-ops/backend/internal/domain/session"
)

type repository interface {
	Create(ctx context.Context, request CreateRequest, target CommandTarget, nextStatus session.Status) (RemoteCommand, error)
	ApplyReceipt(ctx context.Context, request ReceiptRequest, nextStatus *session.Status) (RemoteCommand, error)
	FindTarget(ctx context.Context, sessionID string) (CommandTarget, error)
}

// Service coordinates remote charger command lifecycles.
type Service struct {
	repository repository
}

// NewService creates a remote command service.
func NewService(repository repository) *Service {
	return &Service{repository: repository}
}

// Create validates and records an operations-side remote command.
func (s *Service) Create(ctx context.Context, request CreateRequest) (RemoteCommand, error) {
	request = normalizeCreateRequest(request)
	if err := validateCreateRequest(request); err != nil {
		return RemoteCommand{}, err
	}

	target, err := s.repository.FindTarget(ctx, request.SessionID)
	if err != nil {
		return RemoteCommand{}, err
	}
	if target.ChargerStatus == "offline" || target.ChargerStatus == "disabled" {
		return RemoteCommand{}, ErrChargerOffline
	}

	nextStatus, err := sentTransition(request.CommandType, session.Status(target.SessionStatus))
	if err != nil {
		return RemoteCommand{}, err
	}

	return s.repository.Create(ctx, request, target, nextStatus)
}

// ApplyReceipt records a charger command receipt and advances session state when needed.
func (s *Service) ApplyReceipt(ctx context.Context, request ReceiptRequest) (RemoteCommand, error) {
	request = normalizeReceiptRequest(request)
	if err := validateReceiptRequest(request); err != nil {
		return RemoteCommand{}, err
	}

	nextStatus := acceptedReceiptTransition(request.CommandType, request.Receipt)
	return s.repository.ApplyReceipt(ctx, request, nextStatus)
}

func sentTransition(commandType Type, current session.Status) (session.Status, error) {
	switch commandType {
	case TypeStart:
		return checkedTransition(current, session.StatusStarting)
	case TypeStop:
		return checkedTransition(current, session.StatusStopping)
	case TypePause:
		return requireCurrent(commandType, current, session.StatusCharging)
	case TypeResume:
		return requireCurrent(commandType, current, session.StatusPaused)
	case TypeLimitPower:
		return requireCurrent(commandType, current, session.StatusCharging)
	case TypeReset:
		return requireActive(commandType, current)
	default:
		return "", fmt.Errorf("%w: %s", ErrInvalidCommand, commandType)
	}
}

func requireCurrent(commandType Type, current session.Status, allowed ...session.Status) (session.Status, error) {
	for _, status := range allowed {
		if current == status {
			return current, nil
		}
	}
	return "", fmt.Errorf("%w: %s cannot run while %s", ErrInvalidTransition, commandType, current)
}

func requireActive(commandType Type, current session.Status) (session.Status, error) {
	switch current {
	case session.StatusBilled, session.StatusCancelled:
		return "", fmt.Errorf("%w: %s cannot run while %s", ErrInvalidTransition, commandType, current)
	case "":
		return "", fmt.Errorf("%w: missing session status", ErrInvalidTransition)
	default:
		return current, nil
	}
}

func acceptedReceiptTransition(commandType Type, receipt Status) *session.Status {
	if receipt != StatusAccepted {
		return nil
	}

	var target session.Status
	switch commandType {
	case TypeStart:
		target = session.StatusCharging
	case TypeStop:
		target = session.StatusPendingBilling
	case TypePause:
		target = session.StatusPaused
	case TypeResume:
		target = session.StatusCharging
	default:
		return nil
	}
	return &target
}

func checkedTransition(current session.Status, target session.Status) (session.Status, error) {
	if err := session.ValidateTransition(current, target); err != nil {
		return "", fmt.Errorf("%w: %w", ErrInvalidTransition, err)
	}
	return target, nil
}

func normalizeCreateRequest(request CreateRequest) CreateRequest {
	request.SessionID = strings.TrimSpace(request.SessionID)
	request.RequestedBy = trimDefault(request.RequestedBy, "operations")
	if len(request.Payload) == 0 {
		request.Payload = json.RawMessage(`{}`)
	}
	return request
}

func normalizeReceiptRequest(request ReceiptRequest) ReceiptRequest {
	request.ChargerCode = strings.TrimSpace(request.ChargerCode)
	request.CommandNo = strings.TrimSpace(request.CommandNo)
	request.SessionNo = strings.TrimSpace(request.SessionNo)
	request.Message = strings.TrimSpace(request.Message)
	if len(request.Payload) == 0 {
		request.Payload = json.RawMessage(`{}`)
	}
	return request
}

func validateCreateRequest(request CreateRequest) error {
	if request.SessionID == "" {
		return fmt.Errorf("%w: missing session id", ErrInvalidCommand)
	}
	if !validCommandType(request.CommandType) {
		return fmt.Errorf("%w: %s", ErrInvalidCommand, request.CommandType)
	}
	if request.CommandType == TypeLimitPower && request.TargetPowerKW == nil {
		return fmt.Errorf("%w: missing target power", ErrInvalidCommand)
	}
	return nil
}

func validateReceiptRequest(request ReceiptRequest) error {
	if request.ChargerCode == "" || request.CommandNo == "" {
		return fmt.Errorf("%w: missing charger or command", ErrInvalidReceipt)
	}
	if !validCommandType(request.CommandType) {
		return fmt.Errorf("%w: %s", ErrInvalidReceipt, request.CommandType)
	}
	if !validReceiptStatus(request.Receipt) {
		return fmt.Errorf("%w: %s", ErrInvalidReceipt, request.Receipt)
	}
	return nil
}

func validCommandType(commandType Type) bool {
	switch commandType {
	case TypeStart, TypeStop, TypePause, TypeResume, TypeReset, TypeLimitPower:
		return true
	default:
		return false
	}
}

func validReceiptStatus(status Status) bool {
	switch status {
	case StatusAccepted, StatusRejected, StatusTimeout, StatusFailed:
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
