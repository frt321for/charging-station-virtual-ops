package session

import (
	"encoding/json"
	"time"
)

// Status represents a charging session lifecycle state.
type Status string

const (
	StatusReserved       Status = "reserved"
	StatusWaitingArrival Status = "waiting_arrival"
	StatusPluggedIn      Status = "plugged_in"
	StatusStarting       Status = "starting"
	StatusCharging       Status = "charging"
	StatusPaused         Status = "paused"
	StatusStopping       Status = "stopping"
	StatusPendingBilling Status = "pending_billing"
	StatusBilled         Status = "billed"
	StatusPendingReview  Status = "pending_review"
	StatusCancelled      Status = "cancelled"
)

// Summary is a charging session list row.
type Summary struct {
	ID                string     `json:"id"`
	SessionNo         string     `json:"sessionNo"`
	Status            Status     `json:"status"`
	SiteID            string     `json:"siteId"`
	SiteName          string     `json:"siteName"`
	ChargerID         string     `json:"chargerId"`
	ChargerCode       string     `json:"chargerCode"`
	ConnectorID       string     `json:"connectorId"`
	ConnectorCode     string     `json:"connectorCode"`
	MeterStartKWh     *float64   `json:"meterStartKwh,omitempty"`
	MeterStopKWh      *float64   `json:"meterStopKwh,omitempty"`
	ReservationExpiry *time.Time `json:"reservationExpiry,omitempty"`
	StartedAt         *time.Time `json:"startedAt,omitempty"`
	StoppedAt         *time.Time `json:"stoppedAt,omitempty"`
	StopReason        string     `json:"stopReason"`
	UpdatedAt         time.Time  `json:"updatedAt"`
}

// Detail contains a charging session plus its event timeline and meter samples.
type Detail struct {
	Summary
	Events      []Event      `json:"events"`
	MeterValues []MeterValue `json:"meterValues"`
}

// Event is a business timeline event for a charging session.
type Event struct {
	ID         string          `json:"id"`
	EventType  string          `json:"eventType"`
	OccurredAt time.Time       `json:"occurredAt"`
	Source     string          `json:"source"`
	Payload    json.RawMessage `json:"payload"`
}

// MeterValue is a time-series meter sample for a session.
type MeterValue struct {
	Time       time.Time       `json:"time"`
	PowerKW    float64         `json:"powerKw"`
	VoltageV   *float64        `json:"voltageV,omitempty"`
	CurrentA   *float64        `json:"currentA,omitempty"`
	MeterKWh   float64         `json:"meterKwh"`
	RawPayload json.RawMessage `json:"rawPayload"`
}

// TransitionParams describes a state transition request.
type TransitionParams struct {
	SessionID    string
	TargetStatus Status
	EventType    string
	Source       string
	Payload      json.RawMessage
}
