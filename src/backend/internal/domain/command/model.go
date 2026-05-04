package command

import (
	"encoding/json"
	"time"
)

// Type is the remote command issued to a virtual charger.
type Type string

const (
	TypeStart      Type = "start"
	TypeStop       Type = "stop"
	TypePause      Type = "pause"
	TypeResume     Type = "resume"
	TypeReset      Type = "reset"
	TypeLimitPower Type = "limit_power"
)

// Status is the remote command lifecycle state.
type Status string

const (
	StatusSent     Status = "sent"
	StatusAccepted Status = "accepted"
	StatusRejected Status = "rejected"
	StatusTimeout  Status = "timeout"
	StatusFailed   Status = "failed"
	StatusRetried  Status = "retried"
)

// CreateRequest describes an operations-side command request.
type CreateRequest struct {
	SessionID     string
	CommandType   Type
	RequestedBy   string
	TargetPowerKW *float64
	Payload       json.RawMessage
}

// ReceiptRequest describes a charger-side command receipt.
type ReceiptRequest struct {
	ChargerCode string
	CommandNo   string
	SessionNo   string
	CommandType Type
	Receipt     Status
	Message     string
	Payload     json.RawMessage
}

// RemoteCommand is the persisted command record returned by APIs.
type RemoteCommand struct {
	ID             string          `json:"id"`
	CommandNo      string          `json:"commandNo"`
	SessionID      *string         `json:"sessionId,omitempty"`
	SessionNo      string          `json:"sessionNo"`
	ChargerID      string          `json:"chargerId"`
	ChargerCode    string          `json:"chargerCode"`
	ConnectorID    *string         `json:"connectorId,omitempty"`
	ConnectorCode  string          `json:"connectorCode"`
	CommandType    Type            `json:"commandType"`
	Status         Status          `json:"status"`
	RequestedBy    string          `json:"requestedBy"`
	TargetPowerKW  *float64        `json:"targetPowerKw,omitempty"`
	Payload        json.RawMessage `json:"payload"`
	ResultMessage  string          `json:"resultMessage"`
	SentAt         time.Time       `json:"sentAt"`
	AcknowledgedAt *time.Time      `json:"acknowledgedAt,omitempty"`
}

// CommandTarget is the session and charger target resolved for a command.
type CommandTarget struct {
	SessionID     string
	SessionNo     string
	SiteID        string
	SiteCode      string
	SessionStatus string
	ChargerID     string
	ChargerCode   string
	ConnectorID   string
	ConnectorCode string
	ChargerStatus string
}
