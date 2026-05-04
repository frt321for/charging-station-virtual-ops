package loadcontrol

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"time"

	"charging-ops/backend/internal/platform/database"

	"github.com/jackc/pgx/v5"
)

// Repository persists load policies, queues, and control records.
type Repository struct {
	database *database.Client
}

// NewRepository creates a load-control repository.
func NewRepository(database *database.Client) *Repository {
	return &Repository{database: database}
}

// Snapshot returns the load-control state for a site.
func (r *Repository) Snapshot(ctx context.Context, siteID string) (Snapshot, error) {
	snapshot, err := r.loadSiteSnapshot(ctx, siteID)
	if err != nil {
		return Snapshot{}, err
	}
	policies, err := r.listPolicies(ctx, snapshot.SiteID)
	if err != nil {
		return Snapshot{}, err
	}
	queue, err := r.listQueue(ctx, snapshot.SiteID)
	if err != nil {
		return Snapshot{}, err
	}
	records, err := r.listRecords(ctx, snapshot.SiteID)
	if err != nil {
		return Snapshot{}, err
	}

	snapshot.Policies = policies
	snapshot.Queue = queue
	snapshot.Records = records
	return snapshot, nil
}

// CreateRecord inserts a manual load-control record.
func (r *Repository) CreateRecord(ctx context.Context, params CreateRecordParams) (LoadControlRecord, error) {
	target, err := r.findRecordTarget(ctx, params.SiteID, params.SessionID)
	if err != nil {
		return LoadControlRecord{}, err
	}

	recordNo := "LC-" + time.Now().UTC().Format("20060102150405.000000000")
	var recordID string
	err = r.database.Pool().QueryRow(ctx, `
		INSERT INTO load_control_records (
			record_no, site_id, area_id, group_id, session_id, connector_id,
			action_type, trigger_type, reason, before_load_kw, after_load_kw,
			target_power_kw, status, operator_name
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 'manual', $8, $9, $10, $11, $12, $13)
		RETURNING id::text
	`,
		recordNo,
		target.SiteID,
		target.AreaID,
		target.GroupID,
		target.SessionID,
		target.ConnectorID,
		params.ActionType,
		params.Reason,
		params.BeforeLoadKW,
		params.AfterLoadKW,
		params.TargetPowerKW,
		params.Status,
		params.OperatorName,
	).Scan(&recordID)
	if err != nil {
		return LoadControlRecord{}, fmt.Errorf("insert load-control record: %w", err)
	}
	return r.getRecord(ctx, recordID)
}

// FindRecordTarget resolves the site boundary for a load-control write before persistence.
func (r *Repository) FindRecordTarget(ctx context.Context, siteID string, sessionID string) (recordTarget, error) {
	return r.findRecordTarget(ctx, siteID, sessionID)
}

func (r *Repository) loadSiteSnapshot(ctx context.Context, siteID string) (Snapshot, error) {
	var snapshot Snapshot
	err := r.database.Pool().QueryRow(ctx, `
		WITH selected_site AS (
			SELECT id, code, name, load_limit_kw
			FROM sites
			WHERE deleted_at IS NULL AND ($1 = '' OR id::text = $1 OR code = $1)
			ORDER BY code
			LIMIT 1
		),
		active_sessions AS (
			SELECT cs.id
			FROM charging_sessions cs
			INNER JOIN selected_site s ON s.id = cs.site_id
			WHERE cs.deleted_at IS NULL AND cs.status IN ('starting', 'charging', 'paused', 'stopping')
		),
		latest_meter AS (
			SELECT DISTINCT ON (mv.session_id) mv.session_id, mv.power_kw
			FROM charger_meter_values mv
			INNER JOIN active_sessions a ON a.id = mv.session_id
			ORDER BY mv.session_id, mv.time DESC
		)
		SELECT
			s.id::text,
			s.code,
			s.name,
			COALESCE(SUM(lm.power_kw), 0)::float8,
			s.load_limit_kw
		FROM selected_site s
		LEFT JOIN latest_meter lm ON true
		GROUP BY s.id, s.code, s.name, s.load_limit_kw
	`, siteID).Scan(
		&snapshot.SiteID,
		&snapshot.SiteCode,
		&snapshot.SiteName,
		&snapshot.CurrentLoadKW,
		&snapshot.LoadLimitKW,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Snapshot{}, ErrNotFound
	}
	if err != nil {
		return Snapshot{}, fmt.Errorf("query load snapshot: %w", err)
	}
	snapshot.CurrentLoadKW = round1(snapshot.CurrentLoadKW)
	snapshot.AvailableCapacityKW = round1(math.Max(snapshot.LoadLimitKW-snapshot.CurrentLoadKW, 0))
	return snapshot, nil
}

func (r *Repository) listPolicies(ctx context.Context, siteID string) ([]LoadPolicy, error) {
	rows, err := r.database.Pool().Query(ctx, `
		SELECT id::text, site_id::text, scope_type, scope_code, scope_name,
		       threshold_kw, warning_kw, action_mode, version, status
		FROM load_policies
		WHERE deleted_at IS NULL AND site_id::text = $1
		ORDER BY
			CASE scope_type WHEN 'site' THEN 0 WHEN 'area' THEN 1 ELSE 2 END,
			scope_code,
			version DESC
	`, siteID)
	if err != nil {
		return nil, fmt.Errorf("query load policies: %w", err)
	}
	defer rows.Close()

	items := make([]LoadPolicy, 0)
	for rows.Next() {
		var item LoadPolicy
		if err := rows.Scan(
			&item.ID,
			&item.SiteID,
			&item.ScopeType,
			&item.ScopeCode,
			&item.ScopeName,
			&item.ThresholdKW,
			&item.WarningKW,
			&item.ActionMode,
			&item.Version,
			&item.Status,
		); err != nil {
			return nil, fmt.Errorf("scan load policy: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate load policies: %w", err)
	}
	return items, nil
}

func (r *Repository) listQueue(ctx context.Context, siteID string) ([]QueueItem, error) {
	rows, err := r.database.Pool().Query(ctx, `
		SELECT
			ROW_NUMBER() OVER (ORDER BY cs.reservation_expires_at NULLS LAST, cs.updated_at)::int AS position,
			cs.id::text,
			cs.session_no,
			cn.code,
			cs.status,
			cs.reservation_expires_at,
			GREATEST(0, FLOOR(EXTRACT(EPOCH FROM now() - cs.updated_at) / 60))::int AS wait_minutes,
			CASE
				WHEN cs.reservation_expires_at IS NOT NULL AND cs.reservation_expires_at < now() THEN 'timeout-release'
				ELSE 'capacity-queue'
			END AS reason,
			cs.updated_at
		FROM charging_sessions cs
		INNER JOIN connectors cn ON cn.id = cs.connector_id
		WHERE cs.deleted_at IS NULL
		  AND cs.site_id::text = $1
		  AND cs.status IN ('reserved', 'waiting_arrival')
		ORDER BY cs.reservation_expires_at NULLS LAST, cs.updated_at
		LIMIT 20
	`, siteID)
	if err != nil {
		return nil, fmt.Errorf("query load queue: %w", err)
	}
	defer rows.Close()

	items := make([]QueueItem, 0)
	for rows.Next() {
		var item QueueItem
		var expiry sql.NullTime
		if err := rows.Scan(
			&item.Position,
			&item.SessionID,
			&item.SessionNo,
			&item.ConnectorCode,
			&item.Status,
			&expiry,
			&item.WaitMinutes,
			&item.Reason,
			&item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan load queue: %w", err)
		}
		item.ReservationExpiry = nullableTime(expiry)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate load queue: %w", err)
	}
	return items, nil
}

func (r *Repository) listRecords(ctx context.Context, siteID string) ([]LoadControlRecord, error) {
	rows, err := r.database.Pool().Query(ctx, recordSQL+`
		WHERE lcr.deleted_at IS NULL AND lcr.site_id::text = $1
		ORDER BY lcr.created_at DESC
		LIMIT 20
	`, siteID)
	if err != nil {
		return nil, fmt.Errorf("query load-control records: %w", err)
	}
	defer rows.Close()
	return scanRecords(rows)
}

func (r *Repository) getRecord(ctx context.Context, recordID string) (LoadControlRecord, error) {
	rows, err := r.database.Pool().Query(ctx, recordSQL+`
		WHERE lcr.id::text = $1
	`, recordID)
	if err != nil {
		return LoadControlRecord{}, fmt.Errorf("query load-control record: %w", err)
	}
	defer rows.Close()

	records, err := scanRecords(rows)
	if err != nil {
		return LoadControlRecord{}, err
	}
	if len(records) == 0 {
		return LoadControlRecord{}, ErrNotFound
	}
	return records[0], nil
}

func (r *Repository) findRecordTarget(ctx context.Context, siteID string, sessionID string) (recordTarget, error) {
	if sessionID != "" {
		return r.findSessionRecordTarget(ctx, sessionID)
	}
	var target recordTarget
	err := r.database.Pool().QueryRow(ctx, `
		SELECT id::text, code
		FROM sites
		WHERE deleted_at IS NULL AND (id::text = $1 OR code = $1)
	`, siteID).Scan(&target.SiteID, &target.SiteCode)
	if errors.Is(err, pgx.ErrNoRows) {
		return recordTarget{}, ErrNotFound
	}
	if err != nil {
		return recordTarget{}, fmt.Errorf("query load record site: %w", err)
	}
	return target, nil
}

func (r *Repository) findSessionRecordTarget(ctx context.Context, sessionID string) (recordTarget, error) {
	var target recordTarget
	var areaID, groupID, connectorID sql.NullString
	err := r.database.Pool().QueryRow(ctx, `
		SELECT
			s.id::text,
			s.code,
			a.id::text,
			g.id::text,
			cs.id::text,
			cs.session_no,
			cn.id::text,
			cn.code
		FROM charging_sessions cs
		INNER JOIN sites s ON s.id = cs.site_id
		INNER JOIN chargers c ON c.id = cs.charger_id
		INNER JOIN charger_groups g ON g.id = c.group_id
		INNER JOIN areas a ON a.id = g.area_id
		INNER JOIN connectors cn ON cn.id = cs.connector_id
		WHERE cs.deleted_at IS NULL AND (cs.id::text = $1 OR cs.session_no = $1)
	`, sessionID).Scan(
		&target.SiteID,
		&target.SiteCode,
		&areaID,
		&groupID,
		&target.SessionID,
		&target.SessionNo,
		&connectorID,
		&target.ConnectorCode,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return recordTarget{}, ErrNotFound
	}
	if err != nil {
		return recordTarget{}, fmt.Errorf("query load record session target: %w", err)
	}
	target.AreaID = nullableString(areaID)
	target.GroupID = nullableString(groupID)
	target.ConnectorID = nullableString(connectorID)
	return target, nil
}

const recordSQL = `
	SELECT
		lcr.id::text,
		lcr.record_no,
		lcr.site_id::text,
		s.code,
		COALESCE(g.code, a.code, s.code),
		lcr.session_id::text,
		COALESCE(cs.session_no, ''),
		COALESCE(cn.code, ''),
		lcr.action_type,
		lcr.trigger_type,
		lcr.reason,
		lcr.before_load_kw,
		lcr.after_load_kw,
		lcr.target_power_kw,
		lcr.status,
		lcr.operator_name,
		lcr.created_at
	FROM load_control_records lcr
	INNER JOIN sites s ON s.id = lcr.site_id
	LEFT JOIN areas a ON a.id = lcr.area_id
	LEFT JOIN charger_groups g ON g.id = lcr.group_id
	LEFT JOIN charging_sessions cs ON cs.id = lcr.session_id
	LEFT JOIN connectors cn ON cn.id = lcr.connector_id
`

func scanRecords(rows pgx.Rows) ([]LoadControlRecord, error) {
	records := make([]LoadControlRecord, 0)
	for rows.Next() {
		var record LoadControlRecord
		var sessionID sql.NullString
		var targetPower sql.NullFloat64
		if err := rows.Scan(
			&record.ID,
			&record.RecordNo,
			&record.SiteID,
			&record.SiteCode,
			&record.ScopeCode,
			&sessionID,
			&record.SessionNo,
			&record.ConnectorCode,
			&record.ActionType,
			&record.TriggerType,
			&record.Reason,
			&record.BeforeLoadKW,
			&record.AfterLoadKW,
			&targetPower,
			&record.Status,
			&record.OperatorName,
			&record.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan load-control record: %w", err)
		}
		record.SessionID = nullableString(sessionID)
		record.TargetPowerKW = nullableFloat(targetPower)
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate load-control records: %w", err)
	}
	return records, nil
}

func nullableString(value sql.NullString) *string {
	if !value.Valid || value.String == "" {
		return nil
	}
	return &value.String
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

func round1(value float64) float64 {
	return math.Round(value*10) / 10
}
