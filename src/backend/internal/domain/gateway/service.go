package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type repository interface {
	Register(ctx context.Context, request RegisterRequest) (ChargerSnapshot, error)
	Heartbeat(ctx context.Context, chargerCode string, request HeartbeatRequest) (ChargerSnapshot, error)
	Status(ctx context.Context, chargerCode string, request StatusRequest) (ChargerSnapshot, error)
	MeterValue(ctx context.Context, chargerCode string, request MeterValueRequest) (ChargerSnapshot, error)
	Alarm(ctx context.Context, chargerCode string, request AlarmRequest) (ChargerSnapshot, error)
	Offline(ctx context.Context, chargerCode string) (ChargerSnapshot, error)
}

// Service validates virtual charger access requests.
type Service struct {
	repository repository
}

// NewService creates a gateway service.
func NewService(repository repository) *Service {
	return &Service{repository: repository}
}

// Register creates or refreshes a virtual charger.
func (s *Service) Register(ctx context.Context, request RegisterRequest) (ChargerSnapshot, error) {
	request = normalizeRegister(request)
	if request.GroupCode == "" || request.Code == "" {
		return ChargerSnapshot{}, fmt.Errorf("%w: missing group or charger code", ErrInvalidRequest)
	}
	if request.ChargerType != "ac" && request.ChargerType != "dc" {
		return ChargerSnapshot{}, fmt.Errorf("%w: invalid charger type", ErrInvalidRequest)
	}
	if request.RatedPowerKW <= 0 || request.ConnectorCount <= 0 || request.ConnectorMaxPowerKW <= 0 {
		return ChargerSnapshot{}, fmt.Errorf("%w: invalid charger power or connector count", ErrInvalidRequest)
	}
	return s.repository.Register(ctx, request)
}

// Heartbeat records charger liveness.
func (s *Service) Heartbeat(ctx context.Context, chargerCode string, request HeartbeatRequest) (ChargerSnapshot, error) {
	chargerCode = strings.TrimSpace(chargerCode)
	request.Status = trimDefault(request.Status, "available")
	request.Payload = normalizePayload(request.Payload)
	if chargerCode == "" {
		return ChargerSnapshot{}, fmt.Errorf("%w: missing charger code", ErrInvalidRequest)
	}
	return s.repository.Heartbeat(ctx, chargerCode, request)
}

// Status records charger and connector state.
func (s *Service) Status(ctx context.Context, chargerCode string, request StatusRequest) (ChargerSnapshot, error) {
	chargerCode = strings.TrimSpace(chargerCode)
	request.Status = strings.TrimSpace(request.Status)
	request.SessionNo = strings.TrimSpace(request.SessionNo)
	request.Payload = normalizePayload(request.Payload)
	if chargerCode == "" {
		return ChargerSnapshot{}, fmt.Errorf("%w: missing charger code", ErrInvalidRequest)
	}
	return s.repository.Status(ctx, chargerCode, request)
}

// MeterValue records a time-series meter sample.
func (s *Service) MeterValue(ctx context.Context, chargerCode string, request MeterValueRequest) (ChargerSnapshot, error) {
	chargerCode = strings.TrimSpace(chargerCode)
	request.ConnectorCode = strings.TrimSpace(request.ConnectorCode)
	request.SessionNo = strings.TrimSpace(request.SessionNo)
	request.Payload = normalizePayload(request.Payload)
	if chargerCode == "" || request.ConnectorCode == "" || request.MeterKWh < 0 || request.PowerKW < 0 {
		return ChargerSnapshot{}, fmt.Errorf("%w: invalid meter value", ErrInvalidRequest)
	}
	return s.repository.MeterValue(ctx, chargerCode, request)
}

// Alarm records a charger fault alarm.
func (s *Service) Alarm(ctx context.Context, chargerCode string, request AlarmRequest) (ChargerSnapshot, error) {
	chargerCode = strings.TrimSpace(chargerCode)
	request.ConnectorCode = strings.TrimSpace(request.ConnectorCode)
	request.SessionNo = strings.TrimSpace(request.SessionNo)
	request.FaultCode = strings.TrimSpace(request.FaultCode)
	request.Severity = trimDefault(request.Severity, "medium")
	request.Payload = normalizePayload(request.Payload)
	if chargerCode == "" || request.FaultCode == "" {
		return ChargerSnapshot{}, fmt.Errorf("%w: missing alarm target or fault", ErrInvalidRequest)
	}
	if !validSeverity(request.Severity) {
		return ChargerSnapshot{}, fmt.Errorf("%w: invalid severity", ErrInvalidRequest)
	}
	return s.repository.Alarm(ctx, chargerCode, request)
}

// Offline records charger offline reporting.
func (s *Service) Offline(ctx context.Context, chargerCode string) (ChargerSnapshot, error) {
	chargerCode = strings.TrimSpace(chargerCode)
	if chargerCode == "" {
		return ChargerSnapshot{}, fmt.Errorf("%w: missing charger code", ErrInvalidRequest)
	}
	return s.repository.Offline(ctx, chargerCode)
}

func normalizeRegister(request RegisterRequest) RegisterRequest {
	request.GroupCode = strings.TrimSpace(request.GroupCode)
	request.Code = strings.TrimSpace(request.Code)
	request.Name = trimDefault(request.Name, request.Code)
	request.ChargerType = trimDefault(request.ChargerType, "ac")
	request.InstallationLocation = trimDefault(request.InstallationLocation, "virtual")
	request.Payload = normalizePayload(request.Payload)
	return request
}

func normalizePayload(payload json.RawMessage) json.RawMessage {
	if len(payload) == 0 {
		return json.RawMessage(`{}`)
	}
	return payload
}

func validSeverity(severity string) bool {
	switch severity {
	case "low", "medium", "high", "critical":
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
