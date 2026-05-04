package loadcontrol

import "time"

// Snapshot is the load-control console data for a site.
type Snapshot struct {
	SiteID              string              `json:"siteId"`
	SiteCode            string              `json:"siteCode"`
	SiteName            string              `json:"siteName"`
	CurrentLoadKW       float64             `json:"currentLoadKw"`
	LoadLimitKW         float64             `json:"loadLimitKw"`
	AvailableCapacityKW float64             `json:"availableCapacityKw"`
	Policies            []LoadPolicy        `json:"policies"`
	Queue               []QueueItem         `json:"queue"`
	Records             []LoadControlRecord `json:"records"`
}

// LoadPolicy defines a site, area, or group load boundary.
type LoadPolicy struct {
	ID          string  `json:"id"`
	SiteID      string  `json:"siteId"`
	ScopeType   string  `json:"scopeType"`
	ScopeCode   string  `json:"scopeCode"`
	ScopeName   string  `json:"scopeName"`
	ThresholdKW float64 `json:"thresholdKw"`
	WarningKW   float64 `json:"warningKw"`
	ActionMode  string  `json:"actionMode"`
	Version     int     `json:"version"`
	Status      string  `json:"status"`
}

// QueueItem is a reservation or waiting-arrival item waiting for capacity.
type QueueItem struct {
	Position          int        `json:"position"`
	SessionID         string     `json:"sessionId"`
	SessionNo         string     `json:"sessionNo"`
	ConnectorCode     string     `json:"connectorCode"`
	Status            string     `json:"status"`
	ReservationExpiry *time.Time `json:"reservationExpiry,omitempty"`
	WaitMinutes       int        `json:"waitMinutes"`
	Reason            string     `json:"reason"`
	UpdatedAt         time.Time  `json:"updatedAt"`
}

// LoadControlRecord records queueing, limiting, pause, resume, and rejection decisions.
type LoadControlRecord struct {
	ID            string    `json:"id"`
	RecordNo      string    `json:"recordNo"`
	SiteID        string    `json:"siteId"`
	SiteCode      string    `json:"siteCode"`
	ScopeCode     string    `json:"scopeCode"`
	SessionID     *string   `json:"sessionId,omitempty"`
	SessionNo     string    `json:"sessionNo"`
	ConnectorCode string    `json:"connectorCode"`
	ActionType    string    `json:"actionType"`
	TriggerType   string    `json:"triggerType"`
	Reason        string    `json:"reason"`
	BeforeLoadKW  float64   `json:"beforeLoadKw"`
	AfterLoadKW   float64   `json:"afterLoadKw"`
	TargetPowerKW *float64  `json:"targetPowerKw,omitempty"`
	Status        string    `json:"status"`
	OperatorName  string    `json:"operatorName"`
	CreatedAt     time.Time `json:"createdAt"`
}

// CreateRecordParams describes a manual load-control decision.
type CreateRecordParams struct {
	SiteID        string
	SessionID     string
	ActionType    string
	Reason        string
	BeforeLoadKW  float64
	AfterLoadKW   float64
	TargetPowerKW *float64
	Status        string
	OperatorName  string
}

type recordTarget struct {
	SiteID        string
	SiteCode      string
	AreaID        *string
	GroupID       *string
	SessionID     *string
	SessionNo     string
	ConnectorID   *string
	ConnectorCode string
}
