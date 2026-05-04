package configmgmt

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"charging-ops/backend/internal/domain/auth"
	"charging-ops/backend/internal/platform/database"

	"github.com/jackc/pgx/v5"
)

// Repository reads and updates operations configuration records.
type Repository struct {
	database *database.Client
}

// NewRepository creates a configuration repository.
func NewRepository(database *database.Client) *Repository {
	return &Repository{database: database}
}

// FindSiteBoundary resolves a site UUID/code to its authorization boundary.
func (r *Repository) FindSiteBoundary(ctx context.Context, siteID string) (Boundary, error) {
	var boundary Boundary
	err := r.database.Pool().QueryRow(ctx, `
		SELECT id::text, code
		FROM sites
		WHERE deleted_at IS NULL AND (id::text = $1 OR code = $1)
	`, siteID).Scan(&boundary.SiteID, &boundary.SiteCode)
	if errors.Is(err, pgx.ErrNoRows) {
		return Boundary{}, ErrNotFound
	}
	if err != nil {
		return Boundary{}, fmt.Errorf("query site boundary: %w", err)
	}
	return boundary, nil
}

// FindBoundary resolves a child configuration record to its site boundary.
func (r *Repository) FindBoundary(ctx context.Context, entity string, id string) (Boundary, error) {
	query, ok := boundaryQueries[entity]
	if !ok {
		return Boundary{}, ErrInvalidRequest
	}
	var boundary Boundary
	err := r.database.Pool().QueryRow(ctx, query, id).Scan(&boundary.SiteID, &boundary.SiteCode)
	if errors.Is(err, pgx.ErrNoRows) {
		return Boundary{}, ErrNotFound
	}
	if err != nil {
		return Boundary{}, fmt.Errorf("query %s boundary: %w", entity, err)
	}
	return boundary, nil
}

// Snapshot returns all editable station configuration records.
func (r *Repository) Snapshot(ctx context.Context, siteID string) (Snapshot, error) {
	site, err := r.getSite(ctx, siteID)
	if err != nil {
		return Snapshot{}, err
	}
	areas, err := r.listAreas(ctx, site.ID)
	if err != nil {
		return Snapshot{}, err
	}
	groups, err := r.listGroups(ctx, site.ID)
	if err != nil {
		return Snapshot{}, err
	}
	chargers, err := r.listChargers(ctx, site.ID)
	if err != nil {
		return Snapshot{}, err
	}
	connectors, err := r.listConnectors(ctx, site.ID)
	if err != nil {
		return Snapshot{}, err
	}
	pricingPolicies, err := r.listPricingPolicies(ctx, site.ID)
	if err != nil {
		return Snapshot{}, err
	}
	loadPolicies, err := r.listLoadPolicies(ctx, site.ID)
	if err != nil {
		return Snapshot{}, err
	}
	reservationRules, err := r.listReservationRules(ctx, site.ID)
	if err != nil {
		return Snapshot{}, err
	}
	queueRules, err := r.listQueueRules(ctx, site.ID)
	if err != nil {
		return Snapshot{}, err
	}
	return Snapshot{
		Site:             site,
		Areas:            areas,
		Groups:           groups,
		Chargers:         chargers,
		Connectors:       connectors,
		PricingPolicies:  pricingPolicies,
		LoadPolicies:     loadPolicies,
		ReservationRules: reservationRules,
		QueueRules:       queueRules,
	}, nil
}

func (r *Repository) getSite(ctx context.Context, siteID string) (SiteConfig, error) {
	var item SiteConfig
	err := r.database.Pool().QueryRow(ctx, `
		SELECT
			id::text, code, name, campus, capacity_kw, load_limit_kw, timezone,
			sla_response_minutes, sla_recovery_minutes, status, updated_at
		FROM sites
		WHERE deleted_at IS NULL AND (id::text = $1 OR code = $1)
	`, siteID).Scan(
		&item.ID,
		&item.Code,
		&item.Name,
		&item.Campus,
		&item.CapacityKW,
		&item.LoadLimitKW,
		&item.Timezone,
		&item.SLAResponseMinutes,
		&item.SLARecoveryMinutes,
		&item.Status,
		&item.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return SiteConfig{}, ErrNotFound
	}
	if err != nil {
		return SiteConfig{}, fmt.Errorf("query site config: %w", err)
	}
	return item, nil
}

func (r *Repository) listAreas(ctx context.Context, siteID string) ([]AreaConfig, error) {
	rows, err := r.database.Pool().Query(ctx, `
		SELECT id::text, site_id::text, code, name, load_limit_kw, sort_order, status, updated_at
		FROM areas
		WHERE deleted_at IS NULL AND site_id::text = $1
		ORDER BY sort_order, code
	`, siteID)
	if err != nil {
		return nil, fmt.Errorf("query area config: %w", err)
	}
	defer rows.Close()
	items := make([]AreaConfig, 0)
	for rows.Next() {
		var item AreaConfig
		if err := rows.Scan(&item.ID, &item.SiteID, &item.Code, &item.Name, &item.LoadLimitKW, &item.SortOrder, &item.Status, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan area config: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) listGroups(ctx context.Context, siteID string) ([]GroupConfig, error) {
	rows, err := r.database.Pool().Query(ctx, `
		SELECT id::text, site_id::text, area_id::text, code, name, electrical_node,
		       load_limit_kw, priority, status, updated_at
		FROM charger_groups
		WHERE deleted_at IS NULL AND site_id::text = $1
		ORDER BY priority, code
	`, siteID)
	if err != nil {
		return nil, fmt.Errorf("query group config: %w", err)
	}
	defer rows.Close()
	items := make([]GroupConfig, 0)
	for rows.Next() {
		var item GroupConfig
		if err := rows.Scan(&item.ID, &item.SiteID, &item.AreaID, &item.Code, &item.Name, &item.ElectricalNode, &item.LoadLimitKW, &item.Priority, &item.Status, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan group config: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) listChargers(ctx context.Context, siteID string) ([]ChargerConfig, error) {
	rows, err := r.database.Pool().Query(ctx, `
		SELECT c.id::text, c.group_id::text, g.code, c.code, c.name, c.charger_type,
		       c.rated_power_kw, c.connector_count, c.status, c.installation_location,
		       c.maintenance_tag, c.updated_at
		FROM chargers c
		INNER JOIN charger_groups g ON g.id = c.group_id
		WHERE c.deleted_at IS NULL AND g.site_id::text = $1
		ORDER BY g.priority, c.code
	`, siteID)
	if err != nil {
		return nil, fmt.Errorf("query charger config: %w", err)
	}
	defer rows.Close()
	items := make([]ChargerConfig, 0)
	for rows.Next() {
		var item ChargerConfig
		if err := rows.Scan(&item.ID, &item.GroupID, &item.GroupCode, &item.Code, &item.Name, &item.ChargerType, &item.RatedPowerKW, &item.ConnectorCount, &item.Status, &item.InstallationLocation, &item.MaintenanceTag, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan charger config: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) listConnectors(ctx context.Context, siteID string) ([]ConnectorConfig, error) {
	rows, err := r.database.Pool().Query(ctx, `
		SELECT cn.id::text, cn.charger_id::text, c.code, cn.code, cn.connector_no,
		       cn.max_power_kw, cn.status, cn.updated_at
		FROM connectors cn
		INNER JOIN chargers c ON c.id = cn.charger_id
		INNER JOIN charger_groups g ON g.id = c.group_id
		WHERE cn.deleted_at IS NULL AND c.deleted_at IS NULL AND g.site_id::text = $1
		ORDER BY c.code, cn.connector_no
	`, siteID)
	if err != nil {
		return nil, fmt.Errorf("query connector config: %w", err)
	}
	defer rows.Close()
	items := make([]ConnectorConfig, 0)
	for rows.Next() {
		var item ConnectorConfig
		if err := rows.Scan(&item.ID, &item.ChargerID, &item.ChargerCode, &item.Code, &item.Number, &item.MaxPowerKW, &item.Status, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan connector config: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) listLoadPolicies(ctx context.Context, siteID string) ([]LoadPolicy, error) {
	rows, err := r.database.Pool().Query(ctx, `
		SELECT id::text, site_id::text, scope_type, scope_code, scope_name,
		       threshold_kw, warning_kw, action_mode, version, status, updated_at
		FROM load_policies
		WHERE deleted_at IS NULL AND site_id::text = $1
		ORDER BY scope_type, scope_code, version DESC
	`, siteID)
	if err != nil {
		return nil, fmt.Errorf("query load policies: %w", err)
	}
	defer rows.Close()
	items := make([]LoadPolicy, 0)
	for rows.Next() {
		var item LoadPolicy
		if err := rows.Scan(&item.ID, &item.SiteID, &item.ScopeType, &item.ScopeCode, &item.ScopeName, &item.ThresholdKW, &item.WarningKW, &item.ActionMode, &item.Version, &item.Status, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan load policy: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) listReservationRules(ctx context.Context, siteID string) ([]ReservationRule, error) {
	rows, err := r.database.Pool().Query(ctx, `
		SELECT rr.id::text, rr.site_id::text, s.code, rr.code, rr.name,
		       rr.hold_minutes, rr.timeout_action, rr.version, rr.status, rr.updated_at
		FROM reservation_rules rr
		INNER JOIN sites s ON s.id = rr.site_id
		WHERE rr.deleted_at IS NULL AND rr.site_id::text = $1
		ORDER BY rr.code, rr.version DESC
	`, siteID)
	if err != nil {
		return nil, fmt.Errorf("query reservation rules: %w", err)
	}
	defer rows.Close()
	items := make([]ReservationRule, 0)
	for rows.Next() {
		var item ReservationRule
		if err := rows.Scan(&item.ID, &item.SiteID, &item.SiteCode, &item.Code, &item.Name, &item.HoldMinutes, &item.TimeoutAction, &item.Version, &item.Status, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan reservation rule: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) listQueueRules(ctx context.Context, siteID string) ([]QueueRule, error) {
	rows, err := r.database.Pool().Query(ctx, `
		SELECT qr.id::text, qr.site_id::text, s.code, qr.code, qr.name,
		       qr.strategy, qr.max_queue_size, qr.priority_factor, qr.version, qr.status, qr.updated_at
		FROM queue_rules qr
		INNER JOIN sites s ON s.id = qr.site_id
		WHERE qr.deleted_at IS NULL AND qr.site_id::text = $1
		ORDER BY qr.code, qr.version DESC
	`, siteID)
	if err != nil {
		return nil, fmt.Errorf("query queue rules: %w", err)
	}
	defer rows.Close()
	items := make([]QueueRule, 0)
	for rows.Next() {
		var item QueueRule
		if err := rows.Scan(&item.ID, &item.SiteID, &item.SiteCode, &item.Code, &item.Name, &item.Strategy, &item.MaxQueueSize, &item.PriorityFactor, &item.Version, &item.Status, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan queue rule: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) audit(ctx context.Context, entityType string, entityID string, action string, payload any) error {
	principal, _ := auth.PrincipalFromContext(ctx)
	var userID *string
	if principal.UserID != "" {
		userID = &principal.UserID
	}
	rawPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal config audit payload: %w", err)
	}
	_, err = r.database.Pool().Exec(ctx, `
		INSERT INTO audit_logs (
			entity_type, entity_id, action, actor_user_id, actor_name,
			actor_role_code, trace_id, payload
		)
		VALUES ($1, NULLIF($2, '')::uuid, $3, $4, $5, $6, $7, $8)
	`, entityType, entityID, action, userID, principal.DisplayName, principal.PrimaryRoleCode(), "", rawPayload)
	if err != nil {
		return fmt.Errorf("insert config audit: %w", err)
	}
	return nil
}

func nullableTime(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	return &value.Time
}

var boundaryQueries = map[string]string{
	"area": `
		SELECT s.id::text, s.code
		FROM areas a
		INNER JOIN sites s ON s.id = a.site_id
		WHERE a.deleted_at IS NULL AND a.id::text = $1
	`,
	"group": `
		SELECT s.id::text, s.code
		FROM charger_groups g
		INNER JOIN sites s ON s.id = g.site_id
		WHERE g.deleted_at IS NULL AND g.id::text = $1
	`,
	"charger": `
		SELECT s.id::text, s.code
		FROM chargers c
		INNER JOIN charger_groups g ON g.id = c.group_id
		INNER JOIN sites s ON s.id = g.site_id
		WHERE c.deleted_at IS NULL AND c.id::text = $1
	`,
	"connector": `
		SELECT s.id::text, s.code
		FROM connectors cn
		INNER JOIN chargers c ON c.id = cn.charger_id
		INNER JOIN charger_groups g ON g.id = c.group_id
		INNER JOIN sites s ON s.id = g.site_id
		WHERE cn.deleted_at IS NULL AND cn.id::text = $1
	`,
	"load_policy": `
		SELECT s.id::text, s.code
		FROM load_policies lp
		INNER JOIN sites s ON s.id = lp.site_id
		WHERE lp.deleted_at IS NULL AND lp.id::text = $1
	`,
	"reservation_rule": `
		SELECT s.id::text, s.code
		FROM reservation_rules rr
		INNER JOIN sites s ON s.id = rr.site_id
		WHERE rr.deleted_at IS NULL AND rr.id::text = $1
	`,
	"queue_rule": `
		SELECT s.id::text, s.code
		FROM queue_rules qr
		INNER JOIN sites s ON s.id = qr.site_id
		WHERE qr.deleted_at IS NULL AND qr.id::text = $1
	`,
}
