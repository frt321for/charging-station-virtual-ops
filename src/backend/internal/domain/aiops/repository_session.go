package aiops

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// FindSessionContext loads a session with timeline, commands, readings, bills, and exceptions.
func (r *Repository) FindSessionContext(ctx context.Context, sessionID string) (sessionContext, error) {
	data, err := r.findSessionFact(ctx, sessionID)
	if err != nil {
		return sessionContext{}, err
	}
	events, err := r.listSessionEvents(ctx, data.Session.ID)
	if err != nil {
		return sessionContext{}, err
	}
	meters, err := r.listSessionMeters(ctx, data.Session.ID)
	if err != nil {
		return sessionContext{}, err
	}
	commands, err := r.listSessionCommands(ctx, data.Session.ID)
	if err != nil {
		return sessionContext{}, err
	}
	bills, err := r.listSessionBills(ctx, data.Session.ID)
	if err != nil {
		return sessionContext{}, err
	}
	exceptions, err := r.listSessionExceptions(ctx, data.Session.ID)
	if err != nil {
		return sessionContext{}, err
	}
	data.Events = events
	data.Meters = meters
	data.Commands = commands
	data.Bills = bills
	data.Exceptions = exceptions
	return data, nil
}

func (r *Repository) findSessionFact(ctx context.Context, sessionID string) (sessionContext, error) {
	var data sessionContext
	var startedAt, stoppedAt sql.NullTime
	err := r.database.Pool().QueryRow(ctx, `
		SELECT
			cs.id::text,
			cs.session_no,
			cs.status,
			c.code,
			cn.code,
			cs.started_at,
			cs.stopped_at,
			cs.stop_reason,
			cs.updated_at,
			s.id::text,
			s.code,
			s.name
		FROM charging_sessions cs
		INNER JOIN sites s ON s.id = cs.site_id
		INNER JOIN chargers c ON c.id = cs.charger_id
		INNER JOIN connectors cn ON cn.id = cs.connector_id
		WHERE cs.deleted_at IS NULL AND (cs.id::text = $1 OR cs.session_no = $1)
	`, sessionID).Scan(
		&data.Session.ID,
		&data.Session.SessionNo,
		&data.Session.Status,
		&data.Session.ChargerCode,
		&data.Session.ConnectorCode,
		&startedAt,
		&stoppedAt,
		&data.Session.StopReason,
		&data.Session.UpdatedAt,
		&data.Site.ID,
		&data.Site.Code,
		&data.Site.Name,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return sessionContext{}, ErrNotFound
	}
	if err != nil {
		return sessionContext{}, fmt.Errorf("query ai session context: %w", err)
	}
	data.Session.StartedAt = nullableTime(startedAt)
	data.Session.StoppedAt = nullableTime(stoppedAt)
	return data, nil
}

func (r *Repository) listSessionEvents(ctx context.Context, sessionID string) ([]eventFact, error) {
	rows, err := r.database.Pool().Query(ctx, `
		SELECT event_type, source, occurred_at, payload::text
		FROM session_events
		WHERE session_id::text = $1
		ORDER BY occurred_at DESC, id DESC
		LIMIT 16
	`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("query ai session events: %w", err)
	}
	defer rows.Close()

	items := make([]eventFact, 0)
	for rows.Next() {
		var item eventFact
		if err := rows.Scan(&item.EventType, &item.Source, &item.OccurredAt, &item.Note); err != nil {
			return nil, fmt.Errorf("scan ai session event: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate ai session events: %w", err)
	}
	return items, nil
}

func (r *Repository) listSessionMeters(ctx context.Context, sessionID string) ([]meterFact, error) {
	rows, err := r.database.Pool().Query(ctx, `
		SELECT time, power_kw, meter_kwh
		FROM charger_meter_values
		WHERE session_id::text = $1
		ORDER BY time DESC
		LIMIT 12
	`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("query ai session meter values: %w", err)
	}
	defer rows.Close()

	items := make([]meterFact, 0)
	for rows.Next() {
		var item meterFact
		if err := rows.Scan(&item.Time, &item.PowerKW, &item.MeterKWh); err != nil {
			return nil, fmt.Errorf("scan ai session meter value: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate ai session meter values: %w", err)
	}
	return items, nil
}

func (r *Repository) listSessionCommands(ctx context.Context, sessionID string) ([]commandFact, error) {
	rows, err := r.database.Pool().Query(ctx, `
		SELECT command_no, command_type, status, requested_by, result_message, sent_at, acknowledged_at
		FROM remote_commands
		WHERE deleted_at IS NULL AND session_id::text = $1
		ORDER BY sent_at DESC
		LIMIT 12
	`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("query ai session commands: %w", err)
	}
	defer rows.Close()

	items := make([]commandFact, 0)
	for rows.Next() {
		var item commandFact
		var acknowledgedAt sql.NullTime
		if err := rows.Scan(
			&item.CommandNo,
			&item.CommandType,
			&item.Status,
			&item.RequestedBy,
			&item.ResultMessage,
			&item.SentAt,
			&acknowledgedAt,
		); err != nil {
			return nil, fmt.Errorf("scan ai session command: %w", err)
		}
		item.AcknowledgedAt = nullableTime(acknowledgedAt)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate ai session commands: %w", err)
	}
	return items, nil
}

func (r *Repository) listSessionBills(ctx context.Context, sessionID string) ([]billFact, error) {
	rows, err := r.database.Pool().Query(ctx, `
		SELECT bill_no, status, energy_kwh::float8, duration_minutes, total_amount::float8, exception_flag, generated_at
		FROM billing_drafts
		WHERE deleted_at IS NULL AND session_id::text = $1
		ORDER BY generated_at DESC
	`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("query ai session bills: %w", err)
	}
	defer rows.Close()

	items := make([]billFact, 0)
	for rows.Next() {
		var item billFact
		if err := rows.Scan(
			&item.BillNo,
			&item.Status,
			&item.EnergyKWh,
			&item.DurationMins,
			&item.TotalAmount,
			&item.ExceptionFlag,
			&item.GeneratedAt,
		); err != nil {
			return nil, fmt.Errorf("scan ai session bill: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate ai session bills: %w", err)
	}
	return items, nil
}

func (r *Repository) listSessionExceptions(ctx context.Context, sessionID string) ([]exceptionFact, error) {
	rows, err := r.database.Pool().Query(ctx, `
		SELECT exception_no, exception_type, severity, status, reason, suggested_action, detected_at
		FROM reconciliation_exceptions
		WHERE deleted_at IS NULL AND session_id::text = $1
		ORDER BY detected_at DESC
		LIMIT 12
	`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("query ai session exceptions: %w", err)
	}
	defer rows.Close()

	items := make([]exceptionFact, 0)
	for rows.Next() {
		var item exceptionFact
		if err := rows.Scan(
			&item.ExceptionNo,
			&item.ExceptionType,
			&item.Severity,
			&item.Status,
			&item.Reason,
			&item.SuggestedAction,
			&item.DetectedAt,
		); err != nil {
			return nil, fmt.Errorf("scan ai session exception: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate ai session exceptions: %w", err)
	}
	return items, nil
}
