package gateway

import (
	"context"
	"errors"
	"testing"
)

func TestServiceRegisterValidation(t *testing.T) {
	service := NewService(&fakeRepository{})

	_, err := service.Register(context.Background(), RegisterRequest{
		GroupCode:           "N-A1",
		Code:                "AC-T-001",
		ChargerType:         "ac",
		RatedPowerKW:        7,
		ConnectorCount:      1,
		ConnectorMaxPowerKW: 7,
	})
	if err != nil {
		t.Fatalf("expected valid register request, got %v", err)
	}

	_, err = service.Register(context.Background(), RegisterRequest{
		GroupCode:           "N-A1",
		Code:                "AC-T-001",
		ChargerType:         "bad",
		RatedPowerKW:        7,
		ConnectorCount:      1,
		ConnectorMaxPowerKW: 7,
	})
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("expected invalid request, got %v", err)
	}
}

func TestServiceMeterValueValidation(t *testing.T) {
	service := NewService(&fakeRepository{})

	_, err := service.MeterValue(context.Background(), "AC-T-001", MeterValueRequest{
		ConnectorCode: "AC-T-001-01",
		PowerKW:       -1,
		MeterKWh:      10,
	})
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("expected invalid meter value, got %v", err)
	}
}

type fakeRepository struct{}

func (f *fakeRepository) Register(ctx context.Context, request RegisterRequest) (ChargerSnapshot, error) {
	return ChargerSnapshot{Code: request.Code}, nil
}

func (f *fakeRepository) Heartbeat(ctx context.Context, chargerCode string, request HeartbeatRequest) (ChargerSnapshot, error) {
	return ChargerSnapshot{Code: chargerCode}, nil
}

func (f *fakeRepository) Status(ctx context.Context, chargerCode string, request StatusRequest) (ChargerSnapshot, error) {
	return ChargerSnapshot{Code: chargerCode}, nil
}

func (f *fakeRepository) MeterValue(ctx context.Context, chargerCode string, request MeterValueRequest) (ChargerSnapshot, error) {
	return ChargerSnapshot{Code: chargerCode}, nil
}

func (f *fakeRepository) Alarm(ctx context.Context, chargerCode string, request AlarmRequest) (ChargerSnapshot, error) {
	return ChargerSnapshot{Code: chargerCode}, nil
}

func (f *fakeRepository) Offline(ctx context.Context, chargerCode string) (ChargerSnapshot, error) {
	return ChargerSnapshot{Code: chargerCode}, nil
}
