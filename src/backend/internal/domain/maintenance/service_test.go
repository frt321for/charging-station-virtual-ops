package maintenance

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"charging-ops/backend/internal/domain/auth"
)

func TestValidTransition(t *testing.T) {
	tests := []struct {
		name    string
		current string
		target  string
		want    bool
	}{
		{name: "assign open", current: "open", target: "assigned", want: true},
		{name: "accept assigned", current: "assigned", target: "accepted", want: true},
		{name: "return retest to handling", current: "retest", target: "handling", want: true},
		{name: "close recovered", current: "recovered", target: "closed", want: true},
		{name: "reject skip", current: "open", target: "closed", want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := validTransition(test.current, test.target); got != test.want {
				t.Fatalf("expected %v, got %v", test.want, got)
			}
		})
	}
}

func TestValidStatus(t *testing.T) {
	if !validStatus("handling") {
		t.Fatalf("expected handling to be valid")
	}
	if validStatus("done") {
		t.Fatalf("expected done to be invalid")
	}
}

func TestServiceListEventsRejectsUnauthorizedBeforeRead(t *testing.T) {
	repository := &fakeMaintenanceRepository{
		workOrderBoundary: siteBoundary{SiteID: "site-2", SiteCode: "OTHER"},
	}
	service := NewService(repository)

	_, err := service.ListEvents(siteLimitedContext(), "WO-1")

	if !errors.Is(err, auth.ErrForbidden) {
		t.Fatalf("expected forbidden, got %v", err)
	}
	if repository.listedEvents {
		t.Fatalf("expected no event read before authorization")
	}
}

func TestServiceCreateWorkOrderRejectsUnauthorizedBeforeWrite(t *testing.T) {
	repository := &fakeMaintenanceRepository{
		faultBoundary: siteBoundary{SiteID: "site-2", SiteCode: "OTHER"},
	}
	service := NewService(repository)

	_, err := service.CreateWorkOrder(siteLimitedContext(), CreateWorkOrderParams{
		FaultID:   "FAULT-1",
		ActorName: "maintenance",
	})

	if !errors.Is(err, auth.ErrForbidden) {
		t.Fatalf("expected forbidden, got %v", err)
	}
	if repository.created {
		t.Fatalf("expected no work-order write before authorization")
	}
}

func TestServiceTransitionWorkOrderRejectsUnauthorizedBeforeWrite(t *testing.T) {
	repository := &fakeMaintenanceRepository{
		workOrderBoundary: siteBoundary{SiteID: "site-2", SiteCode: "OTHER"},
	}
	service := NewService(repository)

	_, err := service.TransitionWorkOrder(siteLimitedContext(), TransitionParams{
		WorkOrderID:  "WO-1",
		TargetStatus: "accepted",
		Payload:      json.RawMessage(`{}`),
	})

	if !errors.Is(err, auth.ErrForbidden) {
		t.Fatalf("expected forbidden, got %v", err)
	}
	if repository.transitioned {
		t.Fatalf("expected no work-order transition before authorization")
	}
}

type fakeMaintenanceRepository struct {
	faultBoundary     siteBoundary
	workOrderBoundary siteBoundary
	listedEvents      bool
	created           bool
	transitioned      bool
}

func (f *fakeMaintenanceRepository) Snapshot(ctx context.Context, siteID string) (Snapshot, error) {
	return Snapshot{SLA: SLASummary{SiteID: "site-1", SiteCode: "HQ-CAMPUS"}}, nil
}

func (f *fakeMaintenanceRepository) FindFaultBoundary(ctx context.Context, faultID string) (siteBoundary, error) {
	if f.faultBoundary.SiteID == "" {
		return siteBoundary{SiteID: "site-1", SiteCode: "HQ-CAMPUS"}, nil
	}
	return f.faultBoundary, nil
}

func (f *fakeMaintenanceRepository) FindWorkOrderBoundary(ctx context.Context, workOrderID string) (siteBoundary, error) {
	if f.workOrderBoundary.SiteID == "" {
		return siteBoundary{SiteID: "site-1", SiteCode: "HQ-CAMPUS"}, nil
	}
	return f.workOrderBoundary, nil
}

func (f *fakeMaintenanceRepository) ListEvents(ctx context.Context, workOrderID string) ([]WorkOrderEvent, error) {
	f.listedEvents = true
	return []WorkOrderEvent{}, nil
}

func (f *fakeMaintenanceRepository) CreateWorkOrder(ctx context.Context, params CreateWorkOrderParams) (WorkOrder, error) {
	f.created = true
	return testWorkOrder(), nil
}

func (f *fakeMaintenanceRepository) TransitionWorkOrder(ctx context.Context, params TransitionParams) (WorkOrder, error) {
	f.transitioned = true
	workOrder := testWorkOrder()
	workOrder.Status = params.TargetStatus
	return workOrder, nil
}

func testWorkOrder() WorkOrder {
	now := time.Now().UTC()
	return WorkOrder{
		ID:            "work-order-1",
		WorkOrderNo:   "WO-1",
		SiteID:        "site-1",
		SiteCode:      "HQ-CAMPUS",
		ChargerID:     "charger-1",
		ChargerCode:   "AC-N-001",
		Severity:      "medium",
		Status:        "assigned",
		ResponseDueAt: now.Add(time.Hour),
		RecoveryDueAt: now.Add(2 * time.Hour),
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

func siteLimitedContext() context.Context {
	return auth.WithPrincipal(context.Background(), auth.Principal{
		Permissions: []string{auth.PermissionMaintenance},
		Sites:       []auth.AuthorizedSite{{ID: "site-1", Code: "HQ-CAMPUS"}},
	})
}
