package aiops

import "time"

// InsightKind identifies one AI assistant workflow.
type InsightKind string

const (
	KindSessionExplanation InsightKind = "session_explanation"
	KindWorkOrderSummary   InsightKind = "work_order_summary"
	KindCongestionRisk     InsightKind = "congestion_risk"
	KindDailyReport        InsightKind = "daily_report"
	KindStationQA          InsightKind = "station_qa"
)

// Insight is the read-only AI assistant result returned to the console.
type Insight struct {
	RequestID   string      `json:"requestId"`
	Kind        InsightKind `json:"kind"`
	Site        SiteRef     `json:"site"`
	Target      TargetRef   `json:"target"`
	Content     string      `json:"content"`
	Suggestions []string    `json:"suggestions"`
	Evidence    []Evidence  `json:"evidence"`
	Provider    string      `json:"provider"`
	Model       string      `json:"model"`
	GeneratedAt time.Time   `json:"generatedAt"`
}

// Evidence is a compact source fact used by an AI result.
type Evidence struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

// SiteRef is the authorized site boundary for an AI result.
type SiteRef struct {
	ID   string `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}

// TargetRef identifies the selected session, work order, or site.
type TargetRef struct {
	Type string `json:"type"`
	ID   string `json:"id"`
	Code string `json:"code"`
}

// SessionExplanationParams selects a session for abnormal-chain explanation.
type SessionExplanationParams struct {
	SessionID string
}

// WorkOrderSummaryParams selects a work order for timeline summarization.
type WorkOrderSummaryParams struct {
	WorkOrderID string
}

// CongestionRiskParams selects a site and forecast horizon.
type CongestionRiskParams struct {
	SiteID       string
	HorizonHours int
}

// DailyReportParams selects a site and operating date.
type DailyReportParams struct {
	SiteID       string
	BusinessDate string
}

// StationQAParams asks a bounded station question.
type StationQAParams struct {
	SiteID   string
	Question string
}

type sessionContext struct {
	Site       SiteRef
	Session    sessionFact
	Events     []eventFact
	Meters     []meterFact
	Commands   []commandFact
	Bills      []billFact
	Exceptions []exceptionFact
}

type workOrderContext struct {
	Site      SiteRef
	WorkOrder workOrderFact
	Events    []eventFact
}

type siteContext struct {
	Site                   SiteRef
	CapacityKW             float64
	LoadLimitKW            float64
	CurrentLoadKW          float64
	ActiveSessions         int
	WaitingSessions        int
	AvailableConnectors    int
	FaultedChargers        int
	OpenWorkOrders         int
	OpenExceptions         int
	RevenueToday           float64
	EnergyTodayKWh         float64
	RecentLoadControlCount int
}

type sessionFact struct {
	ID            string     `json:"id"`
	SessionNo     string     `json:"sessionNo"`
	Status        string     `json:"status"`
	ChargerCode   string     `json:"chargerCode"`
	ConnectorCode string     `json:"connectorCode"`
	StartedAt     *time.Time `json:"startedAt,omitempty"`
	StoppedAt     *time.Time `json:"stoppedAt,omitempty"`
	StopReason    string     `json:"stopReason"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}

type eventFact struct {
	EventType  string    `json:"eventType"`
	Source     string    `json:"source"`
	OccurredAt time.Time `json:"occurredAt"`
	Note       string    `json:"note,omitempty"`
}

type meterFact struct {
	Time     time.Time `json:"time"`
	PowerKW  float64   `json:"powerKw"`
	MeterKWh float64   `json:"meterKwh"`
}

type commandFact struct {
	CommandNo      string     `json:"commandNo"`
	CommandType    string     `json:"commandType"`
	Status         string     `json:"status"`
	RequestedBy    string     `json:"requestedBy"`
	ResultMessage  string     `json:"resultMessage"`
	SentAt         time.Time  `json:"sentAt"`
	AcknowledgedAt *time.Time `json:"acknowledgedAt,omitempty"`
}

type billFact struct {
	BillNo        string    `json:"billNo"`
	Status        string    `json:"status"`
	EnergyKWh     float64   `json:"energyKwh"`
	DurationMins  int       `json:"durationMins"`
	TotalAmount   float64   `json:"totalAmount"`
	ExceptionFlag bool      `json:"exceptionFlag"`
	GeneratedAt   time.Time `json:"generatedAt"`
}

type exceptionFact struct {
	ExceptionNo     string    `json:"exceptionNo"`
	ExceptionType   string    `json:"exceptionType"`
	Severity        string    `json:"severity"`
	Status          string    `json:"status"`
	Reason          string    `json:"reason"`
	SuggestedAction string    `json:"suggestedAction"`
	DetectedAt      time.Time `json:"detectedAt"`
}

type workOrderFact struct {
	ID            string    `json:"id"`
	WorkOrderNo   string    `json:"workOrderNo"`
	Status        string    `json:"status"`
	Severity      string    `json:"severity"`
	Title         string    `json:"title"`
	Description   string    `json:"description"`
	AssigneeName  string    `json:"assigneeName"`
	ChargerCode   string    `json:"chargerCode"`
	ConnectorCode string    `json:"connectorCode"`
	SessionNo     string    `json:"sessionNo"`
	ResponseDueAt time.Time `json:"responseDueAt"`
	RecoveryDueAt time.Time `json:"recoveryDueAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}
