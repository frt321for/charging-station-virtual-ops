package gateway

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

type meterTarget struct {
	chargerID   string
	connectorID string
	sessionID   *string
}

func findMeterTarget(ctx context.Context, tx txExecutor, chargerCode string, req MeterValueRequest) (meterTarget, error) {
	var target meterTarget
	var sessionID sql.NullString
	err := tx.QueryRow(ctx, `
		SELECT c.id::text, cn.id::text, cs.id::text
		FROM chargers c
		INNER JOIN connectors cn ON cn.charger_id = c.id
		LEFT JOIN charging_sessions cs ON cs.session_no = $3 AND cs.deleted_at IS NULL
		WHERE c.code = $1 AND cn.code = $2 AND c.deleted_at IS NULL AND cn.deleted_at IS NULL
	`, chargerCode, req.ConnectorCode, req.SessionNo).Scan(&target.chargerID, &target.connectorID, &sessionID)
	if errors.Is(err, pgx.ErrNoRows) {
		return meterTarget{}, ErrNotFound
	}
	if err != nil {
		return meterTarget{}, fmt.Errorf("query meter target: %w", err)
	}
	if sessionID.Valid {
		target.sessionID = &sessionID.String
	}
	return target, nil
}

func insertMeterValue(ctx context.Context, tx txExecutor, target meterTarget, measuredAt time.Time, req MeterValueRequest) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO charger_meter_values (
			time, charger_id, connector_id, session_id, power_kw, voltage_v, current_a, meter_kwh, raw_payload
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, measuredAt, target.chargerID, target.connectorID, target.sessionID, req.PowerKW, req.VoltageV, req.CurrentA, req.MeterKWh, req.Payload)
	if err != nil {
		return fmt.Errorf("insert meter value: %w", err)
	}
	return nil
}

func markMeterTargetCharging(ctx context.Context, tx txExecutor, target meterTarget) error {
	if _, err := tx.Exec(ctx, "UPDATE chargers SET status = 'charging' WHERE id::text = $1", target.chargerID); err != nil {
		return fmt.Errorf("mark charger charging: %w", err)
	}
	if _, err := tx.Exec(ctx, "UPDATE connectors SET status = 'charging' WHERE id::text = $1", target.connectorID); err != nil {
		return fmt.Errorf("mark connector charging: %w", err)
	}
	return nil
}

func insertMeterEvents(ctx context.Context, tx txExecutor, target meterTarget, payload json.RawMessage) error {
	if target.sessionID != nil {
		_, err := tx.Exec(ctx, `
			INSERT INTO session_events (session_id, event_type, source, payload)
			VALUES ($1, 'MeterValueReceived', 'protocol-gateway', $2)
		`, *target.sessionID, payload)
		if err != nil {
			return fmt.Errorf("insert meter session event: %w", err)
		}
	}
	return insertChargerEvent(ctx, tx, target.chargerID, &target.connectorID, target.sessionID, "MeterValueReceived", "protocol-gateway", payload)
}
