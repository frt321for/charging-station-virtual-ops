package gateway

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

type alarmTarget struct {
	chargerID   string
	connectorID *string
	sessionID   *string
}

func findAlarmTarget(ctx context.Context, tx txExecutor, chargerCode string, req AlarmRequest) (alarmTarget, error) {
	var target alarmTarget
	var connectorID, sessionID sql.NullString
	err := tx.QueryRow(ctx, `
		SELECT c.id::text, cn.id::text, cs.id::text
		FROM chargers c
		LEFT JOIN connectors cn ON cn.charger_id = c.id AND cn.code = $2
		LEFT JOIN charging_sessions cs ON cs.session_no = $3 AND cs.deleted_at IS NULL
		WHERE c.code = $1 AND c.deleted_at IS NULL
	`, chargerCode, req.ConnectorCode, req.SessionNo).Scan(&target.chargerID, &connectorID, &sessionID)
	if errors.Is(err, pgx.ErrNoRows) {
		return alarmTarget{}, ErrNotFound
	}
	if err != nil {
		return alarmTarget{}, fmt.Errorf("query alarm target: %w", err)
	}
	target.connectorID = nullableString(connectorID)
	target.sessionID = nullableString(sessionID)
	return target, nil
}

func insertFault(ctx context.Context, tx txExecutor, target alarmTarget, req AlarmRequest) error {
	occurredAt := time.Now().UTC()
	if req.OccurredAt != nil {
		occurredAt = req.OccurredAt.UTC()
	}
	faultNo := "FT-" + time.Now().UTC().Format("20060102150405.000000000")
	_, err := tx.Exec(ctx, `
		INSERT INTO charger_faults (
			fault_no, charger_id, connector_id, session_id, fault_code, severity, occurred_at, payload
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, faultNo, target.chargerID, target.connectorID, target.sessionID, req.FaultCode, req.Severity, occurredAt, req.Payload)
	if err != nil {
		return fmt.Errorf("insert charger fault: %w", err)
	}
	return nil
}

func markFaultStatus(ctx context.Context, tx txExecutor, target alarmTarget) error {
	if _, err := tx.Exec(ctx, "UPDATE chargers SET status = 'fault' WHERE id::text = $1", target.chargerID); err != nil {
		return fmt.Errorf("mark charger fault: %w", err)
	}
	if target.connectorID != nil {
		if _, err := tx.Exec(ctx, "UPDATE connectors SET status = 'fault' WHERE id::text = $1", *target.connectorID); err != nil {
			return fmt.Errorf("mark connector fault: %w", err)
		}
	}
	return nil
}
