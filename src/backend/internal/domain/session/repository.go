package session

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"charging-ops/backend/internal/platform/database"

	"github.com/jackc/pgx/v5"
)

// ErrNotFound is returned when a charging session cannot be found.
var ErrNotFound = errors.New("charging session not found")

// Repository reads and writes charging session facts.
type Repository struct {
	database *database.Client
}

// NewRepository creates a charging session repository.
func NewRepository(database *database.Client) *Repository {
	return &Repository{database: database}
}

// List returns recent charging sessions.
func (r *Repository) List(ctx context.Context) ([]Summary, error) {
	rows, err := r.database.Pool().Query(ctx, sessionSummarySQL+`
		ORDER BY cs.updated_at DESC
		LIMIT 50
	`)
	if err != nil {
		return nil, fmt.Errorf("query charging sessions: %w", err)
	}
	defer rows.Close()

	sessions := make([]Summary, 0)
	for rows.Next() {
		summary, err := scanSummary(rows)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, summary)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate charging sessions: %w", err)
	}

	return sessions, nil
}

// Get returns a charging session detail with timeline and meter samples.
func (r *Repository) Get(ctx context.Context, sessionID string) (Detail, error) {
	summary, err := r.getSummary(ctx, sessionID)
	if err != nil {
		return Detail{}, err
	}

	events, err := r.listEvents(ctx, summary.ID)
	if err != nil {
		return Detail{}, err
	}

	meterValues, err := r.listMeterValues(ctx, summary.ID)
	if err != nil {
		return Detail{}, err
	}

	return Detail{Summary: summary, Events: events, MeterValues: meterValues}, nil
}

// GetStatus returns the current lifecycle status for a session.
func (r *Repository) GetStatus(ctx context.Context, sessionID string) (Status, error) {
	var status Status
	err := r.database.Pool().QueryRow(ctx, `
		SELECT status
		FROM charging_sessions
		WHERE deleted_at IS NULL AND (id::text = $1 OR session_no = $1)
	`, sessionID).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("query charging session status: %w", err)
	}
	return status, nil
}

// AppendTransition updates a session state and appends a timeline event.
func (r *Repository) AppendTransition(ctx context.Context, params TransitionParams) error {
	tx, err := r.database.Pool().Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin session transition: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var sessionID string
	err = tx.QueryRow(ctx, `
		UPDATE charging_sessions
		SET
			status = $2,
			started_at = CASE
				WHEN $2 = 'charging' THEN COALESCE(started_at, now())
				ELSE started_at
			END,
			stopped_at = CASE
				WHEN $2 IN ('pending_billing', 'pending_review', 'billed', 'cancelled') THEN COALESCE(stopped_at, now())
				ELSE stopped_at
			END
		WHERE deleted_at IS NULL AND (id::text = $1 OR session_no = $1)
		RETURNING id::text
	`, params.SessionID, params.TargetStatus).Scan(&sessionID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("update charging session status: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO session_events (session_id, event_type, source, payload)
		VALUES ($1, $2, $3, $4)
	`, sessionID, params.EventType, params.Source, params.Payload); err != nil {
		return fmt.Errorf("insert session event: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit session transition: %w", err)
	}
	return nil
}

const sessionSummarySQL = `
	SELECT
		cs.id::text,
		cs.session_no,
		cs.status,
		s.id::text,
		s.name,
		c.id::text,
		c.code,
		cn.id::text,
		cn.code,
		cs.meter_start_kwh,
		cs.meter_stop_kwh,
		cs.reservation_expires_at,
		cs.started_at,
		cs.stopped_at,
		cs.stop_reason,
		cs.updated_at
	FROM charging_sessions cs
	INNER JOIN sites s ON s.id = cs.site_id
	INNER JOIN chargers c ON c.id = cs.charger_id
	INNER JOIN connectors cn ON cn.id = cs.connector_id
	WHERE cs.deleted_at IS NULL
`

func (r *Repository) getSummary(ctx context.Context, sessionID string) (Summary, error) {
	row := r.database.Pool().QueryRow(ctx, sessionSummarySQL+`
		AND (cs.id::text = $1 OR cs.session_no = $1)
	`, sessionID)

	summary, err := scanSummary(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Summary{}, ErrNotFound
	}
	if err != nil {
		return Summary{}, err
	}
	return summary, nil
}

func (r *Repository) listEvents(ctx context.Context, sessionID string) ([]Event, error) {
	rows, err := r.database.Pool().Query(ctx, `
		SELECT id::text, event_type, occurred_at, source, payload
		FROM session_events
		WHERE session_id::text = $1
		ORDER BY occurred_at, id
	`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("query session events: %w", err)
	}
	defer rows.Close()

	events := make([]Event, 0)
	for rows.Next() {
		var event Event
		var payload []byte
		if err := rows.Scan(&event.ID, &event.EventType, &event.OccurredAt, &event.Source, &payload); err != nil {
			return nil, fmt.Errorf("scan session event: %w", err)
		}
		event.Payload = normalizeJSON(payload)
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate session events: %w", err)
	}

	return events, nil
}

func (r *Repository) listMeterValues(ctx context.Context, sessionID string) ([]MeterValue, error) {
	rows, err := r.database.Pool().Query(ctx, `
		SELECT time, power_kw, voltage_v, current_a, meter_kwh, raw_payload
		FROM charger_meter_values
		WHERE session_id::text = $1
		ORDER BY time
		LIMIT 500
	`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("query session meter values: %w", err)
	}
	defer rows.Close()

	values := make([]MeterValue, 0)
	for rows.Next() {
		var value MeterValue
		var voltage, current sql.NullFloat64
		var payload []byte
		if err := rows.Scan(
			&value.Time,
			&value.PowerKW,
			&voltage,
			&current,
			&value.MeterKWh,
			&payload,
		); err != nil {
			return nil, fmt.Errorf("scan session meter value: %w", err)
		}
		value.VoltageV = nullableFloat(voltage)
		value.CurrentA = nullableFloat(current)
		value.RawPayload = normalizeJSON(payload)
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate session meter values: %w", err)
	}

	return values, nil
}

type summaryScanner interface {
	Scan(dest ...any) error
}

func scanSummary(scanner summaryScanner) (Summary, error) {
	var summary Summary
	var meterStart, meterStop sql.NullFloat64
	var reservationExpiry, startedAt, stoppedAt sql.NullTime
	if err := scanner.Scan(
		&summary.ID,
		&summary.SessionNo,
		&summary.Status,
		&summary.SiteID,
		&summary.SiteName,
		&summary.ChargerID,
		&summary.ChargerCode,
		&summary.ConnectorID,
		&summary.ConnectorCode,
		&meterStart,
		&meterStop,
		&reservationExpiry,
		&startedAt,
		&stoppedAt,
		&summary.StopReason,
		&summary.UpdatedAt,
	); err != nil {
		return Summary{}, fmt.Errorf("scan charging session: %w", err)
	}

	summary.MeterStartKWh = nullableFloat(meterStart)
	summary.MeterStopKWh = nullableFloat(meterStop)
	summary.ReservationExpiry = nullableTime(reservationExpiry)
	summary.StartedAt = nullableTime(startedAt)
	summary.StoppedAt = nullableTime(stoppedAt)
	return summary, nil
}

func nullableFloat(value sql.NullFloat64) *float64 {
	if !value.Valid {
		return nil
	}
	return &value.Float64
}

func nullableTime(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	return &value.Time
}

func normalizeJSON(payload []byte) json.RawMessage {
	if len(payload) == 0 {
		return json.RawMessage(`{}`)
	}
	return json.RawMessage(payload)
}
