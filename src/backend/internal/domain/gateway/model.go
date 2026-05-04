package gateway

import (
	"encoding/json"
	"time"
)

// RegisterRequest creates or refreshes a virtual charger record.
type RegisterRequest struct {
	GroupCode            string
	Code                 string
	Name                 string
	ChargerType          string
	RatedPowerKW         float64
	ConnectorCount       int
	ConnectorMaxPowerKW  float64
	InstallationLocation string
	Payload              json.RawMessage
}

// HeartbeatRequest records charger liveness.
type HeartbeatRequest struct {
	Status  string
	Payload json.RawMessage
}

// StatusRequest records charger and connector status.
type StatusRequest struct {
	Status     string
	SessionNo  string
	Connectors []ConnectorStatus
	Payload    json.RawMessage
}

// ConnectorStatus is a reported connector state.
type ConnectorStatus struct {
	Code   string `json:"code"`
	Number int    `json:"number"`
	Status string `json:"status"`
}

// MeterValueRequest records a charging meter sample.
type MeterValueRequest struct {
	ConnectorCode string
	SessionNo     string
	Time          *time.Time
	PowerKW       float64
	VoltageV      *float64
	CurrentA      *float64
	MeterKWh      float64
	Payload       json.RawMessage
}

// AlarmRequest records a charger-side fault alarm.
type AlarmRequest struct {
	ConnectorCode string
	SessionNo     string
	FaultCode     string
	Severity      string
	OccurredAt    *time.Time
	Payload       json.RawMessage
}

// ChargerSnapshot is returned after gateway writes.
type ChargerSnapshot struct {
	ID              string     `json:"id"`
	Code            string     `json:"code"`
	Name            string     `json:"name"`
	Status          string     `json:"status"`
	ChargerType     string     `json:"chargerType"`
	RatedPowerKW    float64    `json:"ratedPowerKw"`
	ConnectorCount  int        `json:"connectorCount"`
	LastHeartbeatAt *time.Time `json:"lastHeartbeatAt,omitempty"`
}
