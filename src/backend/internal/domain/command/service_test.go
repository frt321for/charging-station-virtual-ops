package command

import (
	"context"
	"errors"
	"testing"

	"charging-ops/backend/internal/domain/session"
)

func TestServiceCreate(t *testing.T) {
	tests := []struct {
		name        string
		command     Type
		status      string
		charger     string
		targetPower *float64
		wantErr     error
	}{
		{name: "start moves plugged session to starting", command: TypeStart, status: "plugged_in", charger: "available"},
		{name: "stop moves charging session to stopping", command: TypeStop, status: "charging", charger: "charging"},
		{name: "pause is allowed while charging", command: TypePause, status: "charging", charger: "charging"},
		{name: "resume is allowed while paused", command: TypeResume, status: "paused", charger: "charging"},
		{name: "limit power is allowed while charging", command: TypeLimitPower, status: "charging", charger: "charging", targetPower: floatPtr(3.5)},
		{name: "limit power requires target", command: TypeLimitPower, status: "charging", charger: "charging", wantErr: ErrInvalidCommand},
		{name: "offline charger is rejected", command: TypeStop, status: "charging", charger: "offline", wantErr: ErrChargerOffline},
		{name: "invalid transition is rejected", command: TypeStart, status: "charging", charger: "charging", wantErr: ErrInvalidTransition},
		{name: "pause is rejected before charging", command: TypePause, status: "plugged_in", charger: "available", wantErr: ErrInvalidTransition},
		{name: "resume is rejected outside paused", command: TypeResume, status: "charging", charger: "charging", wantErr: ErrInvalidTransition},
		{name: "limit power is rejected before charging", command: TypeLimitPower, status: "plugged_in", charger: "available", targetPower: floatPtr(3.5), wantErr: ErrInvalidTransition},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := &fakeRepository{
				target: CommandTarget{
					SessionID:     "session-1",
					SessionStatus: test.status,
					ChargerID:     "charger-1",
					ChargerStatus: test.charger,
					ConnectorID:   "connector-1",
				},
			}
			service := NewService(repository)

			_, err := service.Create(context.Background(), CreateRequest{
				SessionID:     "session-1",
				CommandType:   test.command,
				TargetPowerKW: test.targetPower,
			})

			if test.wantErr == nil && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if test.wantErr != nil && !errors.Is(err, test.wantErr) {
				t.Fatalf("expected %v, got %v", test.wantErr, err)
			}
		})
	}
}

func TestServiceApplyReceipt(t *testing.T) {
	repository := &fakeRepository{}
	service := NewService(repository)

	_, err := service.ApplyReceipt(context.Background(), ReceiptRequest{
		ChargerCode: "AC-N-001",
		CommandNo:   "RC-1",
		CommandType: TypeStart,
		Receipt:     StatusAccepted,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if repository.receiptStatus == nil || *repository.receiptStatus != session.StatusCharging {
		t.Fatalf("expected accepted start receipt to move to charging")
	}
}

type fakeRepository struct {
	target        CommandTarget
	receiptStatus *session.Status
}

func (f *fakeRepository) FindTarget(ctx context.Context, sessionID string) (CommandTarget, error) {
	return f.target, nil
}

func (f *fakeRepository) Create(
	ctx context.Context,
	request CreateRequest,
	target CommandTarget,
	nextStatus session.Status,
) (RemoteCommand, error) {
	return RemoteCommand{CommandType: request.CommandType, Status: StatusSent}, nil
}

func (f *fakeRepository) ApplyReceipt(
	ctx context.Context,
	request ReceiptRequest,
	nextStatus *session.Status,
) (RemoteCommand, error) {
	f.receiptStatus = nextStatus
	return RemoteCommand{CommandNo: request.CommandNo, Status: request.Receipt}, nil
}

func floatPtr(value float64) *float64 {
	return &value
}
