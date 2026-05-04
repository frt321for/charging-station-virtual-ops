package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

func findGroupID(ctx context.Context, tx txExecutor, groupCode string) (string, error) {
	var groupID string
	err := tx.QueryRow(ctx, "SELECT id::text FROM charger_groups WHERE code = $1 AND deleted_at IS NULL", groupCode).Scan(&groupID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("query charger group: %w", err)
	}
	return groupID, nil
}

func upsertCharger(ctx context.Context, tx txExecutor, groupID string, req RegisterRequest) (string, error) {
	var chargerID string
	err := tx.QueryRow(ctx, `
		INSERT INTO chargers (
			group_id, code, name, charger_type, rated_power_kw, connector_count,
			status, installation_location, last_heartbeat_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, 'available', $7, now())
		ON CONFLICT (code) DO UPDATE SET
			name = EXCLUDED.name,
			rated_power_kw = EXCLUDED.rated_power_kw,
			connector_count = EXCLUDED.connector_count,
			status = 'available',
			last_heartbeat_at = now()
		RETURNING id::text
	`, groupID, req.Code, req.Name, req.ChargerType, req.RatedPowerKW, req.ConnectorCount, req.InstallationLocation).Scan(&chargerID)
	if err != nil {
		return "", fmt.Errorf("upsert charger: %w", err)
	}
	return chargerID, nil
}

func upsertConnectors(ctx context.Context, tx txExecutor, chargerID string, req RegisterRequest) error {
	for number := 1; number <= req.ConnectorCount; number++ {
		code := fmt.Sprintf("%s-%02d", req.Code, number)
		_, err := tx.Exec(ctx, `
			INSERT INTO connectors (charger_id, connector_no, code, max_power_kw, status)
			VALUES ($1, $2, $3, $4, 'available')
			ON CONFLICT (code) DO UPDATE SET max_power_kw = EXCLUDED.max_power_kw
		`, chargerID, number, code, req.ConnectorMaxPowerKW)
		if err != nil {
			return fmt.Errorf("upsert connector %d: %w", number, err)
		}
	}
	return nil
}

func updateChargerStatus(ctx context.Context, tx txExecutor, chargerCode string, status string, heartbeat bool) (string, error) {
	if status == "" {
		status = "available"
	}
	sqlText := "UPDATE chargers SET status = $2 WHERE code = $1 AND deleted_at IS NULL RETURNING id::text"
	if heartbeat {
		sqlText = "UPDATE chargers SET status = $2, last_heartbeat_at = now() WHERE code = $1 AND deleted_at IS NULL RETURNING id::text"
	}
	var chargerID string
	err := tx.QueryRow(ctx, sqlText, chargerCode, status).Scan(&chargerID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("update charger status: %w", err)
	}
	return chargerID, nil
}

func findSessionID(ctx context.Context, tx txExecutor, sessionNo string) (*string, error) {
	if sessionNo == "" {
		return nil, nil
	}
	var sessionID string
	err := tx.QueryRow(ctx, "SELECT id::text FROM charging_sessions WHERE session_no = $1 AND deleted_at IS NULL", sessionNo).Scan(&sessionID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query session id: %w", err)
	}
	return &sessionID, nil
}

func updateConnectorStatuses(ctx context.Context, tx txExecutor, chargerID string, sessionID *string, req StatusRequest) error {
	for _, connector := range req.Connectors {
		if connector.Code == "" && connector.Number == 0 {
			continue
		}
		if err := updateConnectorStatus(ctx, tx, chargerID, connector); err != nil {
			return err
		}
		if connector.Status == "plugged" && sessionID != nil {
			if err := plugInSession(ctx, tx, *sessionID, req.Payload); err != nil {
				return err
			}
		}
	}
	return nil
}

func updateConnectorStatus(ctx context.Context, tx txExecutor, chargerID string, connector ConnectorStatus) error {
	if connector.Status == "" {
		return nil
	}
	_, err := tx.Exec(ctx, `
		UPDATE connectors
		SET status = $3
		WHERE charger_id::text = $1 AND (code = $2 OR connector_no = $4)
	`, chargerID, connector.Code, connector.Status, connector.Number)
	if err != nil {
		return fmt.Errorf("update connector status: %w", err)
	}
	return nil
}

func plugInSession(ctx context.Context, tx txExecutor, sessionID string, payload json.RawMessage) error {
	_, err := tx.Exec(ctx, `
		UPDATE charging_sessions
		SET status = 'plugged_in'
		WHERE id::text = $1 AND status IN ('reserved', 'waiting_arrival')
	`, sessionID)
	if err != nil {
		return fmt.Errorf("update session plugged in: %w", err)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO session_events (session_id, event_type, source, payload)
		VALUES ($1, 'PlugInDetected', 'protocol-gateway', $2)
	`, sessionID, payload)
	if err != nil {
		return fmt.Errorf("insert plugin event: %w", err)
	}
	return nil
}
