package maintenance

import (
	"encoding/json"
	"time"
)

// Fault is a detected charger fault with optional work-order linkage.
type Fault struct {
	ID            string     `json:"id"`
	FaultNo       string     `json:"faultNo"`
	SiteID        string     `json:"siteId"`
	SiteCode      string     `json:"siteCode"`
	ChargerID     string     `json:"chargerId"`
	ChargerCode   string     `json:"chargerCode"`
	ConnectorID   *string    `json:"connectorId,omitempty"`
	ConnectorCode string     `json:"connectorCode"`
	SessionID     *string    `json:"sessionId,omitempty"`
	SessionNo     string     `json:"sessionNo"`
	FaultCode     string     `json:"faultCode"`
	Severity      string     `json:"severity"`
	Status        string     `json:"status"`
	WorkOrderID   *string    `json:"workOrderId,omitempty"`
	WorkOrderNo   string     `json:"workOrderNo"`
	RepeatCount   int        `json:"repeatCount"`
	OccurredAt    time.Time  `json:"occurredAt"`
	ResolvedAt    *time.Time `json:"resolvedAt,omitempty"`
}

// WorkOrder describes a maintenance ticket and SLA state.
type WorkOrder struct {
	ID                  string     `json:"id"`
	WorkOrderNo         string     `json:"workOrderNo"`
	FaultID             *string    `json:"faultId,omitempty"`
	FaultNo             string     `json:"faultNo"`
	SiteID              string     `json:"siteId"`
	SiteCode            string     `json:"siteCode"`
	ChargerID           string     `json:"chargerId"`
	ChargerCode         string     `json:"chargerCode"`
	ConnectorID         *string    `json:"connectorId,omitempty"`
	ConnectorCode       string     `json:"connectorCode"`
	SessionID           *string    `json:"sessionId,omitempty"`
	SessionNo           string     `json:"sessionNo"`
	Severity            string     `json:"severity"`
	Status              string     `json:"status"`
	ImpactScope         string     `json:"impactScope"`
	Title               string     `json:"title"`
	Description         string     `json:"description"`
	AssigneeName        string     `json:"assigneeName"`
	ResponseDueAt       time.Time  `json:"responseDueAt"`
	RecoveryDueAt       time.Time  `json:"recoveryDueAt"`
	AcceptedAt          *time.Time `json:"acceptedAt,omitempty"`
	ArrivedAt           *time.Time `json:"arrivedAt,omitempty"`
	HandlingAt          *time.Time `json:"handlingAt,omitempty"`
	RetestAt            *time.Time `json:"retestAt,omitempty"`
	RecoveredAt         *time.Time `json:"recoveredAt,omitempty"`
	ClosedAt            *time.Time `json:"closedAt,omitempty"`
	SLAResponseBreached bool       `json:"slaResponseBreached"`
	SLARecoveryBreached bool       `json:"slaRecoveryBreached"`
	CreatedAt           time.Time  `json:"createdAt"`
	UpdatedAt           time.Time  `json:"updatedAt"`
}

// WorkOrderEvent is an audit trail row for a work order.
type WorkOrderEvent struct {
	ID          string          `json:"id"`
	WorkOrderID string          `json:"workOrderId"`
	EventType   string          `json:"eventType"`
	FromStatus  string          `json:"fromStatus"`
	ToStatus    string          `json:"toStatus"`
	ActorName   string          `json:"actorName"`
	Note        string          `json:"note"`
	Payload     json.RawMessage `json:"payload"`
	OccurredAt  time.Time       `json:"occurredAt"`
}

// SLASummary aggregates open ticket and breach state.
type SLASummary struct {
	SiteID              string  `json:"siteId"`
	SiteCode            string  `json:"siteCode"`
	OpenWorkOrders      int     `json:"openWorkOrders"`
	OverdueResponse     int     `json:"overdueResponse"`
	OverdueRecovery     int     `json:"overdueRecovery"`
	CriticalOpen        int     `json:"criticalOpen"`
	RepeatedFaults      int     `json:"repeatedFaults"`
	AverageRecoveryMins float64 `json:"averageRecoveryMins"`
	ResponseSLAHitRate  float64 `json:"responseSlaHitRate"`
	RecoverySLAHitRate  float64 `json:"recoverySlaHitRate"`
}

// Snapshot returns the maintenance workbench state.
type Snapshot struct {
	Faults     []Fault     `json:"faults"`
	WorkOrders []WorkOrder `json:"workOrders"`
	SLA        SLASummary  `json:"sla"`
}

// CreateWorkOrderParams creates or links a work order from a fault.
type CreateWorkOrderParams struct {
	FaultID      string
	AssigneeName string
	ImpactScope  string
	Title        string
	Description  string
	ActorName    string
}

// TransitionParams advances a work-order lifecycle.
type TransitionParams struct {
	WorkOrderID  string
	TargetStatus string
	AssigneeName string
	ActorName    string
	Note         string
	Payload      json.RawMessage
}
