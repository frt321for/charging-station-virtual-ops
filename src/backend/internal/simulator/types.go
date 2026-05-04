package simulator

import "time"

type registerRequest struct {
	GroupCode            string         `json:"groupCode"`
	Code                 string         `json:"code"`
	Name                 string         `json:"name"`
	ChargerType          string         `json:"chargerType"`
	RatedPowerKW         float64        `json:"ratedPowerKw"`
	ConnectorCount       int            `json:"connectorCount"`
	ConnectorMaxPowerKW  float64        `json:"connectorMaxPowerKw"`
	InstallationLocation string         `json:"installationLocation"`
	Payload              map[string]any `json:"payload"`
}

type heartbeatRequest struct {
	Status  string         `json:"status"`
	Payload map[string]any `json:"payload"`
}

type reservationRequest struct {
	ConnectorCode      string         `json:"connectorCode"`
	ReservationMinutes int            `json:"reservationMinutes"`
	RequestedBy        string         `json:"requestedBy"`
	Payload            map[string]any `json:"payload"`
}

type statusRequest struct {
	Status     string            `json:"status"`
	SessionNo  string            `json:"sessionNo,omitempty"`
	Connectors []connectorStatus `json:"connectors"`
	Payload    map[string]any    `json:"payload"`
}

type connectorStatus struct {
	Code   string `json:"code"`
	Number int    `json:"number,omitempty"`
	Status string `json:"status"`
}

type commandRequest struct {
	RequestedBy   string         `json:"requestedBy"`
	TargetPowerKW *float64       `json:"targetPowerKw,omitempty"`
	Payload       map[string]any `json:"payload"`
}

type receiptRequest struct {
	CommandNo   string         `json:"commandNo"`
	SessionNo   string         `json:"sessionNo"`
	CommandType string         `json:"commandType"`
	Receipt     string         `json:"receipt"`
	Message     string         `json:"message"`
	Payload     map[string]any `json:"payload"`
}

type meterValueRequest struct {
	ConnectorCode string         `json:"connectorCode"`
	SessionNo     string         `json:"sessionNo"`
	Time          time.Time      `json:"time"`
	PowerKW       float64        `json:"powerKw"`
	VoltageV      float64        `json:"voltageV"`
	CurrentA      float64        `json:"currentA"`
	MeterKWh      float64        `json:"meterKwh"`
	Payload       map[string]any `json:"payload"`
}

type alarmRequest struct {
	ConnectorCode string         `json:"connectorCode,omitempty"`
	SessionNo     string         `json:"sessionNo,omitempty"`
	FaultCode     string         `json:"faultCode"`
	Severity      string         `json:"severity"`
	OccurredAt    time.Time      `json:"occurredAt"`
	Payload       map[string]any `json:"payload"`
}

type sessionData struct {
	ID        string `json:"id"`
	SessionNo string `json:"sessionNo"`
	Status    string `json:"status"`
}

type commandData struct {
	CommandNo string `json:"commandNo"`
	Status    string `json:"status"`
}

// SessionResult summarizes one simulated charger lifecycle.
type SessionResult struct {
	ChargerCode   string  `json:"chargerCode"`
	ConnectorCode string  `json:"connectorCode"`
	SessionNo     string  `json:"sessionNo,omitempty"`
	FinalStatus   string  `json:"finalStatus"`
	FaultRaised   bool    `json:"faultRaised"`
	Samples       int     `json:"samples"`
	FinalMeterKWh float64 `json:"finalMeterKwh,omitempty"`
}
