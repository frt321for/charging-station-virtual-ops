package command

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"charging-ops/backend/internal/domain/session"
	"charging-ops/backend/internal/platform/database"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Repository stores remote command lifecycle records.
type Repository struct {
	database *database.Client
}

// NewRepository creates a command repository.
func NewRepository(database *database.Client) *Repository {
	return &Repository{database: database}
}

// FindTarget resolves a charging session into a command target.
func (r *Repository) FindTarget(ctx context.Context, sessionID string) (CommandTarget, error) {
	var target CommandTarget
	err := r.database.Pool().QueryRow(ctx, `
		SELECT
			cs.id::text,
			cs.session_no,
			cs.status,
			c.id::text,
			c.code,
			c.status,
			cn.id::text,
			cn.code
		FROM charging_sessions cs
		INNER JOIN chargers c ON c.id = cs.charger_id
		INNER JOIN connectors cn ON cn.id = cs.connector_id
		WHERE cs.deleted_at IS NULL AND (cs.id::text = $1 OR cs.session_no = $1)
	`, sessionID).Scan(
		&target.SessionID,
		&target.SessionNo,
		&target.SessionStatus,
		&target.ChargerID,
		&target.ChargerCode,
		&target.ChargerStatus,
		&target.ConnectorID,
		&target.ConnectorCode,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return CommandTarget{}, ErrNotFound
	}
	if err != nil {
		return CommandTarget{}, fmt.Errorf("query command target: %w", err)
	}
	return target, nil
}

// Create inserts a remote command and records command-sent events.
func (r *Repository) Create(
	ctx context.Context,
	request CreateRequest,
	target CommandTarget,
	nextStatus session.Status,
) (RemoteCommand, error) {
	tx, err := r.database.Pool().Begin(ctx)
	if err != nil {
		return RemoteCommand{}, fmt.Errorf("begin command create: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	commandNo := "RC-" + time.Now().UTC().Format("20060102150405.000000000")
	var commandID string
	err = tx.QueryRow(ctx, `
		INSERT INTO remote_commands (
			command_no, session_id, charger_id, connector_id, command_type,
			status, requested_by, target_power_kw, payload
		)
		VALUES ($1, $2, $3, $4, $5, 'sent', $6, $7, $8)
		RETURNING id::text
	`,
		commandNo,
		target.SessionID,
		target.ChargerID,
		target.ConnectorID,
		request.CommandType,
		request.RequestedBy,
		request.TargetPowerKW,
		request.Payload,
	).Scan(&commandID)
	if err != nil {
		return RemoteCommand{}, fmt.Errorf("insert remote command: %w", err)
	}

	if err := applySentTransition(ctx, tx, target.SessionID, target.SessionStatus, nextStatus); err != nil {
		return RemoteCommand{}, err
	}
	if err := insertCommandEvents(ctx, tx, commandNo, request, target); err != nil {
		return RemoteCommand{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return RemoteCommand{}, fmt.Errorf("commit command create: %w", err)
	}

	return r.getByID(ctx, commandID)
}

// ApplyReceipt updates a remote command from a charger receipt.
func (r *Repository) ApplyReceipt(
	ctx context.Context,
	request ReceiptRequest,
	nextStatus *session.Status,
) (RemoteCommand, error) {
	tx, err := r.database.Pool().Begin(ctx)
	if err != nil {
		return RemoteCommand{}, fmt.Errorf("begin command receipt: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	target, err := scanReceiptTarget(ctx, tx, request.CommandNo, request.ChargerCode, request.CommandType)
	if err != nil {
		return RemoteCommand{}, err
	}
	if nextStatus != nil {
		if err := updateSessionStatus(ctx, tx, target.sessionID, *nextStatus); err != nil {
			return RemoteCommand{}, err
		}
	}
	if err := updateCommandReceipt(ctx, tx, request); err != nil {
		return RemoteCommand{}, err
	}
	if err := insertReceiptEvents(ctx, tx, request, target); err != nil {
		return RemoteCommand{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return RemoteCommand{}, fmt.Errorf("commit command receipt: %w", err)
	}

	return r.getByCommandNo(ctx, request.CommandNo)
}

func (r *Repository) getByID(ctx context.Context, commandID string) (RemoteCommand, error) {
	return r.scanCommand(ctx, `
		WHERE rc.id::text = $1
	`, commandID)
}

func (r *Repository) getByCommandNo(ctx context.Context, commandNo string) (RemoteCommand, error) {
	return r.scanCommand(ctx, `
		WHERE rc.command_no = $1
	`, commandNo)
}

func (r *Repository) scanCommand(ctx context.Context, whereClause string, arg string) (RemoteCommand, error) {
	var command RemoteCommand
	var sessionID, connectorID sql.NullString
	var targetPower sql.NullFloat64
	var acknowledgedAt sql.NullTime
	var payload []byte

	err := r.database.Pool().QueryRow(ctx, commandSQL+whereClause, arg).Scan(
		&command.ID,
		&command.CommandNo,
		&sessionID,
		&command.SessionNo,
		&command.ChargerID,
		&command.ChargerCode,
		&connectorID,
		&command.ConnectorCode,
		&command.CommandType,
		&command.Status,
		&command.RequestedBy,
		&targetPower,
		&payload,
		&command.ResultMessage,
		&command.SentAt,
		&acknowledgedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return RemoteCommand{}, ErrNotFound
	}
	if err != nil {
		return RemoteCommand{}, fmt.Errorf("query remote command: %w", err)
	}

	command.SessionID = nullableString(sessionID)
	command.ConnectorID = nullableString(connectorID)
	command.TargetPowerKW = nullableFloat(targetPower)
	command.Payload = normalizeJSON(payload)
	command.AcknowledgedAt = nullableTime(acknowledgedAt)
	return command, nil
}

const commandSQL = `
	SELECT
		rc.id::text,
		rc.command_no,
		rc.session_id::text,
		COALESCE(cs.session_no, ''),
		rc.charger_id::text,
		c.code,
		rc.connector_id::text,
		COALESCE(cn.code, ''),
		rc.command_type,
		rc.status,
		rc.requested_by,
		rc.target_power_kw,
		rc.payload,
		rc.result_message,
		rc.sent_at,
		rc.acknowledged_at
	FROM remote_commands rc
	INNER JOIN chargers c ON c.id = rc.charger_id
	LEFT JOIN charging_sessions cs ON cs.id = rc.session_id
	LEFT JOIN connectors cn ON cn.id = rc.connector_id
`

type txExecutor interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type receiptTarget struct {
	commandID   string
	sessionID   string
	chargerID   string
	connectorID *string
}

func scanReceiptTarget(ctx context.Context, tx txExecutor, commandNo string, chargerCode string, commandType Type) (receiptTarget, error) {
	var target receiptTarget
	var sessionID, connectorID sql.NullString
	err := tx.QueryRow(ctx, `
		SELECT rc.id::text, rc.session_id::text, rc.charger_id::text, rc.connector_id::text
		FROM remote_commands rc
		INNER JOIN chargers c ON c.id = rc.charger_id
		WHERE rc.command_no = $1 AND c.code = $2 AND rc.command_type = $3 AND rc.deleted_at IS NULL
	`, commandNo, chargerCode, commandType).Scan(&target.commandID, &sessionID, &target.chargerID, &connectorID)
	if errors.Is(err, pgx.ErrNoRows) {
		return receiptTarget{}, ErrNotFound
	}
	if err != nil {
		return receiptTarget{}, fmt.Errorf("query command receipt target: %w", err)
	}
	target.sessionID = sessionID.String
	target.connectorID = nullableString(connectorID)
	return target, nil
}

func applySentTransition(ctx context.Context, tx txExecutor, sessionID string, current string, next session.Status) error {
	if next == session.Status(current) {
		return nil
	}
	return updateSessionStatus(ctx, tx, sessionID, next)
}

func updateSessionStatus(ctx context.Context, tx txExecutor, sessionID string, status session.Status) error {
	_, err := tx.Exec(ctx, `
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
		WHERE id::text = $1 AND deleted_at IS NULL
	`, sessionID, status)
	if err != nil {
		return fmt.Errorf("update session command status: %w", err)
	}
	return nil
}

func insertCommandEvents(ctx context.Context, tx txExecutor, commandNo string, req CreateRequest, target CommandTarget) error {
	payload := commandPayload(commandNo, string(req.CommandType), req.Payload)
	if err := insertSessionEvent(ctx, tx, target.SessionID, "RemoteCommandSent", "operations", payload); err != nil {
		return err
	}
	return insertChargerEvent(ctx, tx, target.ChargerID, &target.ConnectorID, &target.SessionID, "RemoteCommandSent", "operations", payload)
}

func insertReceiptEvents(ctx context.Context, tx txExecutor, req ReceiptRequest, target receiptTarget) error {
	payload := commandPayload(req.CommandNo, string(req.CommandType), req.Payload)
	if target.sessionID != "" {
		if err := insertSessionEvent(ctx, tx, target.sessionID, "CommandReceiptReceived", "protocol-gateway", payload); err != nil {
			return err
		}
	}
	return insertChargerEvent(ctx, tx, target.chargerID, target.connectorID, nullableStringValue(target.sessionID), "CommandReceiptReceived", "protocol-gateway", payload)
}

func updateCommandReceipt(ctx context.Context, tx txExecutor, req ReceiptRequest) error {
	_, err := tx.Exec(ctx, `
		UPDATE remote_commands
		SET status = $2, result_message = $3, acknowledged_at = now()
		WHERE command_no = $1 AND deleted_at IS NULL
	`, req.CommandNo, req.Receipt, req.Message)
	if err != nil {
		return fmt.Errorf("update command receipt: %w", err)
	}
	return nil
}

func insertSessionEvent(ctx context.Context, tx txExecutor, sessionID string, eventType string, source string, payload json.RawMessage) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO session_events (session_id, event_type, source, payload)
		VALUES ($1, $2, $3, $4)
	`, sessionID, eventType, source, payload)
	if err != nil {
		return fmt.Errorf("insert session command event: %w", err)
	}
	return nil
}

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
		return fmt.Errorf("insert charger command event: %w", err)
	}
	return nil
}

func commandPayload(commandNo string, commandType string, payload json.RawMessage) json.RawMessage {
	value := map[string]any{
		"commandNo":   commandNo,
		"commandType": commandType,
		"payload":     json.RawMessage(payload),
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return encoded
}

func nullableString(value sql.NullString) *string {
	if !value.Valid || value.String == "" {
		return nil
	}
	return &value.String
}

func nullableStringValue(value string) *string {
	if value == "" {
		return nil
	}
	return &value
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
