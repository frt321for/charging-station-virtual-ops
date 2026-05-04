package simcontrol

import (
	"time"

	coresim "charging-ops/backend/internal/simulator"
)

// Request describes a browser-triggered simulator run.
type Request struct {
	ChargerCount   int     `json:"chargerCount"`
	ConnectorCount int     `json:"connectorCount"`
	OnlineRate     float64 `json:"onlineRate"`
	FaultRate      float64 `json:"faultRate"`
	LoadCurve      string  `json:"loadCurve"`
}

// Status describes the current simulator controller state.
type Status struct {
	Running       bool                    `json:"running"`
	Mode          string                  `json:"mode"`
	Config        Request                 `json:"config"`
	StartedAt     *time.Time              `json:"startedAt,omitempty"`
	LastRunAt     *time.Time              `json:"lastRunAt,omitempty"`
	LastStoppedAt *time.Time              `json:"lastStoppedAt,omitempty"`
	LastError     string                  `json:"lastError,omitempty"`
	Generated     int                     `json:"generated"`
	Results       []coresim.SessionResult `json:"results"`
}
