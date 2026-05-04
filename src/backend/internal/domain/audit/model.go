package audit

import (
	"encoding/json"
	"time"
)

// Log is one permission-sensitive audit row.
type Log struct {
	ID            string          `json:"id"`
	EntityType    string          `json:"entityType"`
	EntityID      *string         `json:"entityId,omitempty"`
	Action        string          `json:"action"`
	ActorUserID   *string         `json:"actorUserId,omitempty"`
	ActorName     string          `json:"actorName"`
	ActorRoleCode string          `json:"actorRoleCode"`
	IPAddress     string          `json:"ipAddress"`
	TraceID       string          `json:"traceId"`
	Payload       json.RawMessage `json:"payload"`
	CreatedAt     time.Time       `json:"createdAt"`
}

// ListParams describes audit query filters.
type ListParams struct {
	EntityType string
	Action     string
	ActorName  string
	Page       int
	PageSize   int
}

// Page is a paginated audit response.
type Page struct {
	List       []Log      `json:"list"`
	Pagination Pagination `json:"pagination"`
}

// Pagination describes page state.
type Pagination struct {
	Page       int `json:"page"`
	PageSize   int `json:"pageSize"`
	Total      int `json:"total"`
	TotalPages int `json:"totalPages"`
}
