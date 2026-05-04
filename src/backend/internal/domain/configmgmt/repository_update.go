package configmgmt

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// UpdateSite updates station-wide operational configuration.
func (r *Repository) UpdateSite(ctx context.Context, siteID string, params SiteUpdate) (SiteConfig, error) {
	var item SiteConfig
	err := r.database.Pool().QueryRow(ctx, `
		UPDATE sites
		SET
			name = COALESCE($2, name),
			campus = COALESCE($3, campus),
			capacity_kw = COALESCE($4, capacity_kw),
			load_limit_kw = COALESCE($5, load_limit_kw),
			timezone = COALESCE($6, timezone),
			sla_response_minutes = COALESCE($7, sla_response_minutes),
			sla_recovery_minutes = COALESCE($8, sla_recovery_minutes),
			status = COALESCE($9, status)
		WHERE deleted_at IS NULL AND (id::text = $1 OR code = $1)
		RETURNING id::text, code, name, campus, capacity_kw, load_limit_kw, timezone,
		          sla_response_minutes, sla_recovery_minutes, status, updated_at
	`, siteID, stringValue(params.Name), stringValue(params.Campus), floatValue(params.CapacityKW), floatValue(params.LoadLimitKW), stringValue(params.Timezone), intValue(params.SLAResponseMinutes), intValue(params.SLARecoveryMinutes), stringValue(params.Status)).Scan(
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
	if err != nil {
		return SiteConfig{}, updateError("site", err)
	}
	if err := r.audit(ctx, "site", item.ID, "config.update_site", params); err != nil {
		return SiteConfig{}, err
	}
	return item, nil
}

func (r *Repository) UpdateArea(ctx context.Context, id string, params AreaUpdate) (AreaConfig, error) {
	var item AreaConfig
	err := r.database.Pool().QueryRow(ctx, `
		UPDATE areas
		SET name = COALESCE($2, name),
		    load_limit_kw = COALESCE($3, load_limit_kw),
		    sort_order = COALESCE($4, sort_order),
		    status = COALESCE($5, status)
		WHERE deleted_at IS NULL AND id::text = $1
		RETURNING id::text, site_id::text, code, name, load_limit_kw, sort_order, status, updated_at
	`, id, stringValue(params.Name), floatValue(params.LoadLimitKW), intValue(params.SortOrder), stringValue(params.Status)).Scan(&item.ID, &item.SiteID, &item.Code, &item.Name, &item.LoadLimitKW, &item.SortOrder, &item.Status, &item.UpdatedAt)
	if err != nil {
		return AreaConfig{}, updateError("area", err)
	}
	if err := r.audit(ctx, "area", item.ID, "config.update_area", params); err != nil {
		return AreaConfig{}, err
	}
	return item, nil
}

func (r *Repository) UpdateGroup(ctx context.Context, id string, params GroupUpdate) (GroupConfig, error) {
	var item GroupConfig
	err := r.database.Pool().QueryRow(ctx, `
		UPDATE charger_groups
		SET name = COALESCE($2, name),
		    electrical_node = COALESCE($3, electrical_node),
		    load_limit_kw = COALESCE($4, load_limit_kw),
		    priority = COALESCE($5, priority),
		    status = COALESCE($6, status)
		WHERE deleted_at IS NULL AND id::text = $1
		RETURNING id::text, site_id::text, area_id::text, code, name, electrical_node,
		          load_limit_kw, priority, status, updated_at
	`, id, stringValue(params.Name), stringValue(params.ElectricalNode), floatValue(params.LoadLimitKW), intValue(params.Priority), stringValue(params.Status)).Scan(&item.ID, &item.SiteID, &item.AreaID, &item.Code, &item.Name, &item.ElectricalNode, &item.LoadLimitKW, &item.Priority, &item.Status, &item.UpdatedAt)
	if err != nil {
		return GroupConfig{}, updateError("group", err)
	}
	if err := r.audit(ctx, "charger_group", item.ID, "config.update_group", params); err != nil {
		return GroupConfig{}, err
	}
	return item, nil
}

func (r *Repository) UpdateCharger(ctx context.Context, id string, params ChargerUpdate) (ChargerConfig, error) {
	var item ChargerConfig
	err := r.database.Pool().QueryRow(ctx, `
		UPDATE chargers
		SET name = COALESCE($2, name),
		    rated_power_kw = COALESCE($3, rated_power_kw),
		    status = COALESCE($4, status),
		    installation_location = COALESCE($5, installation_location),
		    maintenance_tag = COALESCE($6, maintenance_tag)
		WHERE deleted_at IS NULL AND id::text = $1
		RETURNING id::text, group_id::text, code, name, charger_type, rated_power_kw,
		          connector_count, status, installation_location, maintenance_tag, updated_at
	`, id, stringValue(params.Name), floatValue(params.RatedPowerKW), stringValue(params.Status), stringValue(params.InstallationLocation), stringValue(params.MaintenanceTag)).Scan(&item.ID, &item.GroupID, &item.Code, &item.Name, &item.ChargerType, &item.RatedPowerKW, &item.ConnectorCount, &item.Status, &item.InstallationLocation, &item.MaintenanceTag, &item.UpdatedAt)
	if err != nil {
		return ChargerConfig{}, updateError("charger", err)
	}
	if err := r.database.Pool().QueryRow(ctx, "SELECT code FROM charger_groups WHERE id::text = $1", item.GroupID).Scan(&item.GroupCode); err != nil {
		return ChargerConfig{}, fmt.Errorf("query updated charger group: %w", err)
	}
	if err := r.audit(ctx, "charger", item.ID, "config.update_charger", params); err != nil {
		return ChargerConfig{}, err
	}
	return item, nil
}

func (r *Repository) UpdateConnector(ctx context.Context, id string, params ConnectorUpdate) (ConnectorConfig, error) {
	var item ConnectorConfig
	err := r.database.Pool().QueryRow(ctx, `
		UPDATE connectors
		SET max_power_kw = COALESCE($2, max_power_kw),
		    status = COALESCE($3, status)
		WHERE deleted_at IS NULL AND id::text = $1
		RETURNING id::text, charger_id::text, code, connector_no, max_power_kw, status, updated_at
	`, id, floatValue(params.MaxPowerKW), stringValue(params.Status)).Scan(&item.ID, &item.ChargerID, &item.Code, &item.Number, &item.MaxPowerKW, &item.Status, &item.UpdatedAt)
	if err != nil {
		return ConnectorConfig{}, updateError("connector", err)
	}
	if err := r.database.Pool().QueryRow(ctx, "SELECT code FROM chargers WHERE id::text = $1", item.ChargerID).Scan(&item.ChargerCode); err != nil {
		return ConnectorConfig{}, fmt.Errorf("query updated connector charger: %w", err)
	}
	if err := r.audit(ctx, "connector", item.ID, "config.update_connector", params); err != nil {
		return ConnectorConfig{}, err
	}
	return item, nil
}

func (r *Repository) UpdateLoadPolicy(ctx context.Context, id string, params LoadPolicyUpdate) (LoadPolicy, error) {
	var item LoadPolicy
	err := r.database.Pool().QueryRow(ctx, `
		UPDATE load_policies
		SET threshold_kw = COALESCE($2, threshold_kw),
		    warning_kw = COALESCE($3, warning_kw),
		    action_mode = COALESCE($4, action_mode),
		    status = COALESCE($5, status)
		WHERE deleted_at IS NULL AND id::text = $1
		RETURNING id::text, site_id::text, scope_type, scope_code, scope_name,
		          threshold_kw, warning_kw, action_mode, version, status, updated_at
	`, id, floatValue(params.ThresholdKW), floatValue(params.WarningKW), stringValue(params.ActionMode), stringValue(params.Status)).Scan(&item.ID, &item.SiteID, &item.ScopeType, &item.ScopeCode, &item.ScopeName, &item.ThresholdKW, &item.WarningKW, &item.ActionMode, &item.Version, &item.Status, &item.UpdatedAt)
	if err != nil {
		return LoadPolicy{}, updateError("load policy", err)
	}
	if err := r.audit(ctx, "load_policy", item.ID, "config.update_load_policy", params); err != nil {
		return LoadPolicy{}, err
	}
	return item, nil
}

func (r *Repository) UpdateReservationRule(ctx context.Context, id string, params ReservationRuleUpdate) (ReservationRule, error) {
	var item ReservationRule
	err := r.database.Pool().QueryRow(ctx, `
		UPDATE reservation_rules
		SET name = COALESCE($2, name),
		    hold_minutes = COALESCE($3, hold_minutes),
		    timeout_action = COALESCE($4, timeout_action),
		    status = COALESCE($5, status)
		WHERE deleted_at IS NULL AND id::text = $1
		RETURNING id::text, site_id::text, code, name, hold_minutes,
		          timeout_action, version, status, updated_at
	`, id, stringValue(params.Name), intValue(params.HoldMinutes), stringValue(params.TimeoutAction), stringValue(params.Status)).Scan(&item.ID, &item.SiteID, &item.Code, &item.Name, &item.HoldMinutes, &item.TimeoutAction, &item.Version, &item.Status, &item.UpdatedAt)
	if err != nil {
		return ReservationRule{}, updateError("reservation rule", err)
	}
	if err := r.database.Pool().QueryRow(ctx, "SELECT code FROM sites WHERE id::text = $1", item.SiteID).Scan(&item.SiteCode); err != nil {
		return ReservationRule{}, fmt.Errorf("query reservation site code: %w", err)
	}
	if err := r.audit(ctx, "reservation_rule", item.ID, "config.update_reservation_rule", params); err != nil {
		return ReservationRule{}, err
	}
	return item, nil
}

func (r *Repository) UpdateQueueRule(ctx context.Context, id string, params QueueRuleUpdate) (QueueRule, error) {
	var item QueueRule
	err := r.database.Pool().QueryRow(ctx, `
		UPDATE queue_rules
		SET name = COALESCE($2, name),
		    strategy = COALESCE($3, strategy),
		    max_queue_size = COALESCE($4, max_queue_size),
		    priority_factor = COALESCE($5, priority_factor),
		    status = COALESCE($6, status)
		WHERE deleted_at IS NULL AND id::text = $1
		RETURNING id::text, site_id::text, code, name, strategy,
		          max_queue_size, priority_factor, version, status, updated_at
	`, id, stringValue(params.Name), stringValue(params.Strategy), intValue(params.MaxQueueSize), stringValue(params.PriorityFactor), stringValue(params.Status)).Scan(&item.ID, &item.SiteID, &item.Code, &item.Name, &item.Strategy, &item.MaxQueueSize, &item.PriorityFactor, &item.Version, &item.Status, &item.UpdatedAt)
	if err != nil {
		return QueueRule{}, updateError("queue rule", err)
	}
	if err := r.database.Pool().QueryRow(ctx, "SELECT code FROM sites WHERE id::text = $1", item.SiteID).Scan(&item.SiteCode); err != nil {
		return QueueRule{}, fmt.Errorf("query queue site code: %w", err)
	}
	if err := r.audit(ctx, "queue_rule", item.ID, "config.update_queue_rule", params); err != nil {
		return QueueRule{}, err
	}
	return item, nil
}

func updateError(entity string, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return fmt.Errorf("update %s config: %w", entity, err)
}

func stringValue(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

func floatValue(value *float64) any {
	if value == nil {
		return nil
	}
	return *value
}

func intValue(value *int) any {
	if value == nil {
		return nil
	}
	return *value
}
