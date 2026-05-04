package session

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestServiceTransition(t *testing.T) {
	tests := []struct {
		name       string
		current    Status
		target     Status
		wantErr    error
		wantAppend bool
	}{
		{
			name:       "persists valid transition",
			current:    StatusReserved,
			target:     StatusWaitingArrival,
			wantAppend: true,
		},
		{
			name:    "rejects invalid transition",
			current: StatusBilled,
			target:  StatusCharging,
			wantErr: ErrInvalidTransition,
		},
		{
			name:    "rejects invalid target",
			current: StatusReserved,
			target:  Status("bad"),
			wantErr: ErrInvalidStatus,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := &fakeRepository{status: test.current}
			service := NewService(repository)

			_, err := service.Transition(context.Background(), TransitionParams{
				SessionID:    "session-1",
				TargetStatus: test.target,
				Payload:      json.RawMessage(`{"source":"test"}`),
			})

			if test.wantErr == nil && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if test.wantErr != nil && !errors.Is(err, test.wantErr) {
				t.Fatalf("expected %v, got %v", test.wantErr, err)
			}
			if repository.appended != test.wantAppend {
				t.Fatalf("expected appended=%v, got %v", test.wantAppend, repository.appended)
			}
		})
	}
}

type fakeRepository struct {
	status   Status
	appended bool
}

func (f *fakeRepository) List(ctx context.Context) ([]Summary, error) {
	return []Summary{}, nil
}

func (f *fakeRepository) Get(ctx context.Context, sessionID string) (Detail, error) {
	return Detail{
		Summary: Summary{
			ID:        sessionID,
			SessionNo: "CS-TEST",
			Status:    f.status,
			UpdatedAt: time.Now(),
		},
	}, nil
}

func (f *fakeRepository) GetStatus(ctx context.Context, sessionID string) (Status, error) {
	return f.status, nil
}

func (f *fakeRepository) AppendTransition(ctx context.Context, params TransitionParams) error {
	f.appended = true
	f.status = params.TargetStatus
	return nil
}
