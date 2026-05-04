package gateway

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"charging-ops/backend/internal/platform/database"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Repository stores charger access events and time-series samples.
type Repository struct {
	database *database.Client
}

// NewRepository creates a gateway repository.
func NewRepository(database *database.Client) *Repository {
	return &Repository{database: database}
}

// Register creates or updates a charger and its connector records.
func (r *Repository) Register(ctx context.Context, request RegisterRequest) (ChargerSnapshot, error) {
	tx, err := r.database.Pool().Begin(ctx)
	if err != nil {
		return ChargerSnapshot{}, fmt.Errorf("begin charger register: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	groupID, err := findGroupID(ctx, tx, request.GroupCode)
	if err != nil {
		return ChargerSnapshot{}, err
	}
	chargerID, err := upsertCharger(ctx, tx, groupID, request)
	if err != nil {
		return ChargerSnapshot{}, err
	}
	if err := upsertConnectors(ctx, tx, chargerID, request); err != nil {
		return ChargerSnapshot{}, err
	}
	if err := insertChargerEvent(ctx, tx, chargerID, nil, nil, "ChargerRegistered", "protocol-gateway", request.Payload); err != nil {
		return ChargerSnapshot{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ChargerSnapshot{}, fmt.Errorf("commit charger register: %w", err)
	}
	return r.GetSnapshot(ctx, request.Code)
}

// Heartbeat records charger liveness and status.
func (r *Repository) Heartbeat(ctx context.Context, chargerCode string, request HeartbeatRequest) (ChargerSnapshot, error) {
	tx, err := r.database.Pool().Begin(ctx)
	if err != nil {
		return ChargerSnapshot{}, fmt.Errorf("begin heartbeat: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	chargerID, err := updateChargerStatus(ctx, tx, chargerCode, request.Status, true)
	if err != nil {
		return ChargerSnapshot{}, err
	}
	if err := insertChargerEvent(ctx, tx, chargerID, nil, nil, "HeartbeatReceived", "protocol-gateway", request.Payload); err != nil {
		return ChargerSnapshot{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ChargerSnapshot{}, fmt.Errorf("commit heartbeat: %w", err)
	}
	return r.GetSnapshot(ctx, chargerCode)
}

// Status records charger and connector state reports.
func (r *Repository) Status(ctx context.Context, chargerCode string, request StatusRequest) (ChargerSnapshot, error) {
	tx, err := r.database.Pool().Begin(ctx)
	if err != nil {
		return ChargerSnapshot{}, fmt.Errorf("begin status report: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	chargerID, err := updateChargerStatus(ctx, tx, chargerCode, request.Status, false)
	if err != nil {
		return ChargerSnapshot{}, err
	}
	sessionID, err := findSessionID(ctx, tx, request.SessionNo)
	if err != nil {
		return ChargerSnapshot{}, err
	}
	if err := updateConnectorStatuses(ctx, tx, chargerID, sessionID, request); err != nil {
		return ChargerSnapshot{}, err
	}
	if err := insertChargerEvent(ctx, tx, chargerID, nil, sessionID, "StatusReported", "protocol-gateway", request.Payload); err != nil {
		return ChargerSnapshot{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ChargerSnapshot{}, fmt.Errorf("commit status report: %w", err)
	}
	return r.GetSnapshot(ctx, chargerCode)
}

// MeterValue records one charger meter value.
func (r *Repository) MeterValue(ctx context.Context, chargerCode string, request MeterValueRequest) (ChargerSnapshot, error) {
	tx, err := r.database.Pool().Begin(ctx)
	if err != nil {
		return ChargerSnapshot{}, fmt.Errorf("begin meter value: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	target, err := findMeterTarget(ctx, tx, chargerCode, request)
	if err != nil {
		return ChargerSnapshot{}, err
	}
	measuredAt := time.Now().UTC()
	if request.Time != nil {
		measuredAt = request.Time.UTC()
	}
	if err := insertMeterValue(ctx, tx, target, measuredAt, request); err != nil {
		return ChargerSnapshot{}, err
	}
	if err := markMeterTargetCharging(ctx, tx, target); err != nil {
		return ChargerSnapshot{}, err
	}
	if err := insertMeterEvents(ctx, tx, target, request.Payload); err != nil {
		return ChargerSnapshot{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ChargerSnapshot{}, fmt.Errorf("commit meter value: %w", err)
	}
	return r.GetSnapshot(ctx, chargerCode)
}

// Alarm records a charger fault.
func (r *Repository) Alarm(ctx context.Context, chargerCode string, request AlarmRequest) (ChargerSnapshot, error) {
	tx, err := r.database.Pool().Begin(ctx)
	if err != nil {
		return ChargerSnapshot{}, fmt.Errorf("begin alarm: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	target, err := findAlarmTarget(ctx, tx, chargerCode, request)
	if err != nil {
		return ChargerSnapshot{}, err
	}
	if err := insertFault(ctx, tx, target, request); err != nil {
		return ChargerSnapshot{}, err
	}
	if err := markFaultStatus(ctx, tx, target); err != nil {
		return ChargerSnapshot{}, err
	}
	if err := insertChargerEvent(ctx, tx, target.chargerID, target.connectorID, target.sessionID, "AlarmReported", "protocol-gateway", request.Payload); err != nil {
		return ChargerSnapshot{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ChargerSnapshot{}, fmt.Errorf("commit alarm: %w", err)
	}
	return r.GetSnapshot(ctx, chargerCode)
}

// Offline marks a charger and all connectors offline.
func (r *Repository) Offline(ctx context.Context, chargerCode string) (ChargerSnapshot, error) {
	tx, err := r.database.Pool().Begin(ctx)
	if err != nil {
		return ChargerSnapshot{}, fmt.Errorf("begin offline report: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	chargerID, err := updateChargerStatus(ctx, tx, chargerCode, "offline", false)
	if err != nil {
		return ChargerSnapshot{}, err
	}
	if _, err := tx.Exec(ctx, "UPDATE connectors SET status = 'offline' WHERE charger_id::text = $1", chargerID); err != nil {
		return ChargerSnapshot{}, fmt.Errorf("mark connectors offline: %w", err)
	}
	if err := insertOfflineFault(ctx, tx, chargerID); err != nil {
		return ChargerSnapshot{}, err
	}
	if err := insertChargerEvent(ctx, tx, chargerID, nil, nil, "OfflineReported", "protocol-gateway", json.RawMessage(`{}`)); err != nil {
		return ChargerSnapshot{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ChargerSnapshot{}, fmt.Errorf("commit offline report: %w", err)
	}
	return r.GetSnapshot(ctx, chargerCode)
}

// GetSnapshot returns a charger snapshot by code.
func (r *Repository) GetSnapshot(ctx context.Context, chargerCode string) (ChargerSnapshot, error) {
	var snapshot ChargerSnapshot
	var heartbeat sql.NullTime
	err := r.database.Pool().QueryRow(ctx, `
		SELECT id::text, code, name, status, charger_type, rated_power_kw, connector_count, last_heartbeat_at
		FROM chargers
		WHERE code = $1 AND deleted_at IS NULL
	`, chargerCode).Scan(
		&snapshot.ID,
		&snapshot.Code,
		&snapshot.Name,
		&snapshot.Status,
		&snapshot.ChargerType,
		&snapshot.RatedPowerKW,
		&snapshot.ConnectorCount,
		&heartbeat,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return ChargerSnapshot{}, ErrNotFound
	}
	if err != nil {
		return ChargerSnapshot{}, fmt.Errorf("query charger snapshot: %w", err)
	}
	if heartbeat.Valid {
		snapshot.LastHeartbeatAt = &heartbeat.Time
	}
	return snapshot, nil
}

type txExecutor interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}
