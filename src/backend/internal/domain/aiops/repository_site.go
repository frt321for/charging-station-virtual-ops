package aiops

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// FindSiteContext loads a compact operational snapshot for one authorized site.
func (r *Repository) FindSiteContext(ctx context.Context, siteID string) (siteContext, error) {
	var data siteContext
	err := r.database.Pool().QueryRow(ctx, `
		WITH target_site AS (
			SELECT id, code, name, capacity_kw, load_limit_kw
			FROM sites
			WHERE deleted_at IS NULL AND ($1 = '' OR id::text = $1 OR code = $1)
			ORDER BY code
			LIMIT 1
		),
		latest_meter AS (
			SELECT DISTINCT ON (cmv.connector_id) cmv.connector_id, cmv.power_kw
			FROM charger_meter_values cmv
			INNER JOIN chargers c ON c.id = cmv.charger_id
			INNER JOIN charger_groups g ON g.id = c.group_id
			WHERE g.site_id = (SELECT id FROM target_site)
			  AND cmv.time >= now() - interval '20 minutes'
			ORDER BY cmv.connector_id, cmv.time DESC
		)
		SELECT
			s.id::text,
			s.code,
			s.name,
			s.capacity_kw,
			s.load_limit_kw,
			COALESCE((SELECT SUM(power_kw)::float8 FROM latest_meter), 0),
			COUNT(DISTINCT cs.id) FILTER (WHERE cs.status IN ('starting', 'charging', 'paused'))::int,
			COUNT(DISTINCT cs.id) FILTER (WHERE cs.status IN ('reserved', 'waiting_arrival'))::int,
			COUNT(DISTINCT cn.id) FILTER (WHERE cn.status = 'available' AND cn.deleted_at IS NULL)::int,
			COUNT(DISTINCT c.id) FILTER (WHERE c.status IN ('fault', 'offline') AND c.deleted_at IS NULL)::int,
			COALESCE((
				SELECT COUNT(*)::int
				FROM work_orders wo
				WHERE wo.site_id = s.id
				  AND wo.deleted_at IS NULL
				  AND wo.status NOT IN ('closed', 'cancelled')
			), 0),
			COALESCE((
				SELECT COUNT(*)::int
				FROM reconciliation_exceptions re
				INNER JOIN charging_sessions rcs ON rcs.id = re.session_id
				WHERE rcs.site_id = s.id
				  AND re.deleted_at IS NULL
				  AND re.status <> 'resolved'
			), 0),
			COALESCE((
				SELECT SUM(bd.total_amount)::float8
				FROM billing_drafts bd
				INNER JOIN charging_sessions bcs ON bcs.id = bd.session_id
				WHERE bcs.site_id = s.id
				  AND bd.deleted_at IS NULL
				  AND bd.generated_at >= date_trunc('day', now())
			), 0),
			COALESCE((
				SELECT SUM(bd.energy_kwh)::float8
				FROM billing_drafts bd
				INNER JOIN charging_sessions bcs ON bcs.id = bd.session_id
				WHERE bcs.site_id = s.id
				  AND bd.deleted_at IS NULL
				  AND bd.generated_at >= date_trunc('day', now())
			), 0),
			COALESCE((
				SELECT COUNT(*)::int
				FROM load_control_records lcr
				WHERE lcr.site_id = s.id
				  AND lcr.deleted_at IS NULL
				  AND lcr.created_at >= now() - interval '24 hours'
			), 0)
		FROM target_site s
		LEFT JOIN charger_groups g ON g.site_id = s.id AND g.deleted_at IS NULL
		LEFT JOIN chargers c ON c.group_id = g.id AND c.deleted_at IS NULL
		LEFT JOIN connectors cn ON cn.charger_id = c.id AND cn.deleted_at IS NULL
		LEFT JOIN charging_sessions cs ON cs.site_id = s.id AND cs.deleted_at IS NULL
		GROUP BY s.id, s.code, s.name, s.capacity_kw, s.load_limit_kw
	`, siteID).Scan(
		&data.Site.ID,
		&data.Site.Code,
		&data.Site.Name,
		&data.CapacityKW,
		&data.LoadLimitKW,
		&data.CurrentLoadKW,
		&data.ActiveSessions,
		&data.WaitingSessions,
		&data.AvailableConnectors,
		&data.FaultedChargers,
		&data.OpenWorkOrders,
		&data.OpenExceptions,
		&data.RevenueToday,
		&data.EnergyTodayKWh,
		&data.RecentLoadControlCount,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return siteContext{}, ErrNotFound
	}
	if err != nil {
		return siteContext{}, fmt.Errorf("query ai site context: %w", err)
	}
	return data, nil
}
