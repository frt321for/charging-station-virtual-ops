package gateway

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

func insertChargerEvent(
	ctx context.Context,
	tx txExecutor,
	chargerID string,
	connectorID *string,
	sessionID *string,
	eventType string,
	source string,
	payload json.RawMessage,
) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO charger_events (charger_id, connector_id, session_id, event_type, source, payload)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, chargerID, connectorID, sessionID, eventType, source, payload)
	if err != nil {
		return fmt.Errorf("insert charger event: %w", err)
	}
	return nil
}

func nullableString(value sql.NullString) *string {
	if !value.Valid || value.String == "" {
		return nil
	}
	return &value.String
}
