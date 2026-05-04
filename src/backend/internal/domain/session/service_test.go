package session

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"charging-ops/backend/internal/domain/auth"
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

func TestServiceTransitionRejectsUnauthorizedBeforeAppend(t *testing.T) {
	repository := &fakeRepository{
		status:   StatusReserved,
		boundary: siteBoundary{SiteID: "site-2", SiteCode: "OTHER"},
	}
	service := NewService(repository)

	_, err := service.Transition(siteLimitedContext(), TransitionParams{
		SessionID:    "session-1",
		TargetStatus: StatusWaitingArrival,
	})

	if !errors.Is(err, auth.ErrForbidden) {
		t.Fatalf("expected forbidden, got %v", err)
	}
	if repository.appended {
		t.Fatalf("expected no write before authorization")
	}
}

func TestServiceCreateReservationRejectsUnauthorizedBeforeCreate(t *testing.T) {
	repository := &fakeRepository{
		connectorBoundary: siteBoundary{SiteID: "site-2", SiteCode: "OTHER"},
	}
	service := NewService(repository)

	_, err := service.CreateReservation(siteLimitedContext(), CreateReservationParams{
		ConnectorCode: "AC-N-001-01",
	})

	if !errors.Is(err, auth.ErrForbidden) {
		t.Fatalf("expected forbidden, got %v", err)
	}
	if repository.created {
		t.Fatalf("expected no reservation write before authorization")
	}
}

type fakeRepository struct {
	status            Status
	boundary          siteBoundary
	connectorBoundary siteBoundary
	appended          bool
	created           bool
}

func (f *fakeRepository) List(ctx context.Context) ([]Summary, error) {
	return []Summary{}, nil
}

func (f *fakeRepository) Get(ctx context.Context, sessionID string) (Detail, error) {
	boundary := f.boundary
	if boundary.SiteID == "" {
		boundary = siteBoundary{SiteID: "site-1", SiteCode: "HQ-CAMPUS"}
	}
	return Detail{
		Summary: Summary{
			ID:        sessionID,
			SessionNo: "CS-TEST",
			Status:    f.status,
			SiteID:    boundary.SiteID,
			SiteName:  boundary.SiteCode,
			UpdatedAt: time.Now(),
		},
	}, nil
}

func (f *fakeRepository) GetBoundary(ctx context.Context, sessionID string) (siteBoundary, error) {
	if f.boundary.SiteID == "" {
		return siteBoundary{SiteID: "site-1", SiteCode: "HQ-CAMPUS"}, nil
	}
	return f.boundary, nil
}

func (f *fakeRepository) GetStatus(ctx context.Context, sessionID string) (Status, error) {
	return f.status, nil
}

func (f *fakeRepository) GetConnectorBoundary(ctx context.Context, connectorCode string) (siteBoundary, error) {
	if f.connectorBoundary.SiteID == "" {
		return siteBoundary{SiteID: "site-1", SiteCode: "HQ-CAMPUS"}, nil
	}
	return f.connectorBoundary, nil
}

func (f *fakeRepository) CreateReservation(ctx context.Context, params CreateReservationParams) (string, error) {
	f.created = true
	f.status = StatusWaitingArrival
	return "CS-TEST", nil
}

func (f *fakeRepository) AppendTransition(ctx context.Context, params TransitionParams) error {
	f.appended = true
	f.status = params.TargetStatus
	return nil
}

func siteLimitedContext() context.Context {
	return auth.WithPrincipal(context.Background(), auth.Principal{
		Permissions: []string{auth.PermissionSessionsWrite},
		Sites:       []auth.AuthorizedSite{{ID: "site-1", Code: "HQ-CAMPUS"}},
	})
}
