package maintenance

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"charging-ops/backend/internal/platform/database"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Repository persists faults, work orders, and SLA state.
type Repository struct {
	database *database.Client
}

// NewRepository creates a maintenance repository.
func NewRepository(database *database.Client) *Repository {
	return &Repository{database: database}
}

// Snapshot returns fault, work-order, and SLA state for one site.
func (r *Repository) Snapshot(ctx context.Context, siteID string) (Snapshot, error) {
	site, err := r.findSite(ctx, siteID)
	if err != nil {
		return Snapshot{}, err
	}
	faults, err := r.ListFaults(ctx, site.ID)
	if err != nil {
		return Snapshot{}, err
	}
	workOrders, err := r.ListWorkOrders(ctx, site.ID)
	if err != nil {
		return Snapshot{}, err
	}
	sla, err := r.SLASummary(ctx, site.ID)
	if err != nil {
		return Snapshot{}, err
	}
	return Snapshot{Faults: faults, WorkOrders: workOrders, SLA: sla}, nil
}

// FindFaultBoundary returns the site boundary for one fault.
func (r *Repository) FindFaultBoundary(ctx context.Context, faultID string) (siteBoundary, error) {
	var boundary siteBoundary
	err := r.database.Pool().QueryRow(ctx, `
		SELECT s.id::text, s.code
		FROM charger_faults cf
		INNER JOIN chargers c ON c.id = cf.charger_id
		INNER JOIN charger_groups g ON g.id = c.group_id
		INNER JOIN sites s ON s.id = g.site_id
		WHERE cf.deleted_at IS NULL AND (cf.id::text = $1 OR cf.fault_no = $1)
	`, faultID).Scan(&boundary.SiteID, &boundary.SiteCode)
	if errors.Is(err, pgx.ErrNoRows) {
		return siteBoundary{}, ErrNotFound
	}
	if err != nil {
		return siteBoundary{}, fmt.Errorf("query fault boundary: %w", err)
	}
	return boundary, nil
}

// FindWorkOrderBoundary returns the site boundary for one work order.
func (r *Repository) FindWorkOrderBoundary(ctx context.Context, workOrderID string) (siteBoundary, error) {
	var boundary siteBoundary
	err := r.database.Pool().QueryRow(ctx, `
		SELECT s.id::text, s.code
		FROM work_orders wo
		INNER JOIN sites s ON s.id = wo.site_id
		WHERE wo.deleted_at IS NULL AND (wo.id::text = $1 OR wo.work_order_no = $1)
	`, workOrderID).Scan(&boundary.SiteID, &boundary.SiteCode)
	if errors.Is(err, pgx.ErrNoRows) {
		return siteBoundary{}, ErrNotFound
	}
	if err != nil {
		return siteBoundary{}, fmt.Errorf("query work-order boundary: %w", err)
	}
	return boundary, nil
}

// ListFaults returns recent open and linked faults.
func (r *Repository) ListFaults(ctx context.Context, siteID string) ([]Fault, error) {
	rows, err := r.database.Pool().Query(ctx, faultSQL+`
		WHERE cf.deleted_at IS NULL
		  AND g.site_id::text = $1
		  AND cf.status <> 'resolved'
		ORDER BY cf.occurred_at DESC
		LIMIT 50
	`, siteID)
	if err != nil {
		return nil, fmt.Errorf("query faults: %w", err)
	}
	defer rows.Close()
	return scanFaults(rows)
}

// ListWorkOrders returns recent maintenance tickets.
func (r *Repository) ListWorkOrders(ctx context.Context, siteID string) ([]WorkOrder, error) {
	rows, err := r.database.Pool().Query(ctx, workOrderSQL+`
		WHERE wo.deleted_at IS NULL
		  AND wo.site_id::text = $1
		ORDER BY
		  CASE wo.status
		    WHEN 'open' THEN 0 WHEN 'assigned' THEN 1 WHEN 'accepted' THEN 2
		    WHEN 'arrived' THEN 3 WHEN 'handling' THEN 4 WHEN 'retest' THEN 5
		    WHEN 'recovered' THEN 6 ELSE 7
		  END,
		  wo.created_at DESC
		LIMIT 50
	`, siteID)
	if err != nil {
		return nil, fmt.Errorf("query work orders: %w", err)
	}
	defer rows.Close()
	return scanWorkOrders(rows)
}

// ListEvents returns the audit timeline for one work order.
func (r *Repository) ListEvents(ctx context.Context, workOrderID string) ([]WorkOrderEvent, error) {
	rows, err := r.database.Pool().Query(ctx, `
		SELECT id::text, work_order_id::text, event_type, from_status, to_status,
		       actor_name, note, payload, occurred_at
		FROM work_order_events
		WHERE work_order_id::text = $1
		ORDER BY occurred_at, id
	`, workOrderID)
	if err != nil {
		return nil, fmt.Errorf("query work-order events: %w", err)
	}
	defer rows.Close()

	events := make([]WorkOrderEvent, 0)
	for rows.Next() {
		var event WorkOrderEvent
		var payload []byte
		if err := rows.Scan(
			&event.ID,
			&event.WorkOrderID,
			&event.EventType,
			&event.FromStatus,
			&event.ToStatus,
			&event.ActorName,
			&event.Note,
			&payload,
			&event.OccurredAt,
		); err != nil {
			return nil, fmt.Errorf("scan work-order event: %w", err)
		}
		event.Payload = normalizeJSON(payload)
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate work-order events: %w", err)
	}
	return events, nil
}

// CreateWorkOrder creates a ticket for a fault, or returns the existing linked ticket.
func (r *Repository) CreateWorkOrder(ctx context.Context, params CreateWorkOrderParams) (WorkOrder, error) {
	tx, err := r.database.Pool().Begin(ctx)
	if err != nil {
		return WorkOrder{}, fmt.Errorf("begin work order: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	target, err := findFaultTarget(ctx, tx, params.FaultID)
	if err != nil {
		return WorkOrder{}, err
	}
	if target.WorkOrderID != "" {
		return r.getWorkOrder(ctx, target.WorkOrderID)
	}

	workOrderID, err := insertWorkOrder(ctx, tx, target, params)
	if err != nil {
		return WorkOrder{}, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE charger_faults
		SET status = 'linked_work_order'
		WHERE id::text = $1 AND deleted_at IS NULL
	`, target.FaultID); err != nil {
		return WorkOrder{}, fmt.Errorf("link fault work order: %w", err)
	}
	if err := insertWorkOrderEvent(ctx, tx, workOrderID, "WorkOrderCreated", "", targetInitialStatus(params), params.ActorName, params.Description, json.RawMessage(`{}`)); err != nil {
		return WorkOrder{}, err
	}
	if err := insertAudit(ctx, tx, "work_order", workOrderID, "create", params.ActorName, json.RawMessage(`{}`)); err != nil {
		return WorkOrder{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return WorkOrder{}, fmt.Errorf("commit work order: %w", err)
	}
	return r.getWorkOrder(ctx, workOrderID)
}

// TransitionWorkOrder advances a ticket and records audit events.
func (r *Repository) TransitionWorkOrder(ctx context.Context, params TransitionParams) (WorkOrder, error) {
	tx, err := r.database.Pool().Begin(ctx)
	if err != nil {
		return WorkOrder{}, fmt.Errorf("begin work-order transition: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	current, err := currentWorkOrderStatus(ctx, tx, params.WorkOrderID)
	if err != nil {
		return WorkOrder{}, err
	}
	if !validTransition(current, params.TargetStatus) {
		return WorkOrder{}, fmt.Errorf("%w: %s to %s", ErrInvalidWorkOrder, current, params.TargetStatus)
	}
	if err := updateWorkOrderStatus(ctx, tx, params, current); err != nil {
		return WorkOrder{}, err
	}
	if err := insertWorkOrderEvent(ctx, tx, params.WorkOrderID, "WorkOrderTransitioned", current, params.TargetStatus, params.ActorName, params.Note, params.Payload); err != nil {
		return WorkOrder{}, err
	}
	if params.TargetStatus == "closed" {
		if err := resolveLinkedFault(ctx, tx, params.WorkOrderID); err != nil {
			return WorkOrder{}, err
		}
	}
	if err := insertAudit(ctx, tx, "work_order", params.WorkOrderID, "transition", params.ActorName, params.Payload); err != nil {
		return WorkOrder{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return WorkOrder{}, fmt.Errorf("commit work-order transition: %w", err)
	}
	return r.getWorkOrder(ctx, params.WorkOrderID)
}

// SLASummary returns aggregate SLA state for one site.
func (r *Repository) SLASummary(ctx context.Context, siteID string) (SLASummary, error) {
	var summary SLASummary
	var average sql.NullFloat64
	err := r.database.Pool().QueryRow(ctx, `
		WITH base AS (
			SELECT wo.*
			FROM work_orders wo
			WHERE wo.deleted_at IS NULL AND wo.site_id::text = $1
		),
		repeated AS (
			SELECT COUNT(*)::int AS repeated_count
			FROM (
				SELECT charger_id
				FROM charger_faults
				WHERE deleted_at IS NULL
				  AND charger_id IN (SELECT charger_id FROM base)
				  AND occurred_at >= now() - interval '7 days'
				GROUP BY charger_id
				HAVING COUNT(*) > 1
			) grouped
		)
		SELECT
			s.id::text,
			s.code,
			COUNT(*) FILTER (WHERE b.status NOT IN ('closed', 'cancelled'))::int,
			COUNT(*) FILTER (
				WHERE b.status NOT IN ('accepted', 'arrived', 'handling', 'retest', 'recovered', 'closed', 'cancelled')
				  AND now() > b.response_due_at
			)::int,
			COUNT(*) FILTER (
				WHERE b.status NOT IN ('recovered', 'closed', 'cancelled')
				  AND now() > b.recovery_due_at
			)::int,
			COUNT(*) FILTER (WHERE b.status NOT IN ('closed', 'cancelled') AND b.severity = 'critical')::int,
			COALESCE((SELECT repeated_count FROM repeated), 0),
			AVG(EXTRACT(EPOCH FROM (b.recovered_at - b.created_at)) / 60) FILTER (WHERE b.recovered_at IS NOT NULL),
			COALESCE(100.0 * COUNT(*) FILTER (WHERE b.accepted_at IS NOT NULL AND b.accepted_at <= b.response_due_at) / NULLIF(COUNT(*) FILTER (WHERE b.accepted_at IS NOT NULL), 0), 100)::float8,
			COALESCE(100.0 * COUNT(*) FILTER (WHERE b.recovered_at IS NOT NULL AND b.recovered_at <= b.recovery_due_at) / NULLIF(COUNT(*) FILTER (WHERE b.recovered_at IS NOT NULL), 0), 100)::float8
		FROM sites s
		LEFT JOIN base b ON b.site_id = s.id
		WHERE s.id::text = $1
		GROUP BY s.id, s.code
	`, siteID).Scan(
		&summary.SiteID,
		&summary.SiteCode,
		&summary.OpenWorkOrders,
		&summary.OverdueResponse,
		&summary.OverdueRecovery,
		&summary.CriticalOpen,
		&summary.RepeatedFaults,
		&average,
		&summary.ResponseSLAHitRate,
		&summary.RecoverySLAHitRate,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return SLASummary{}, ErrNotFound
	}
	if err != nil {
		return SLASummary{}, fmt.Errorf("query SLA summary: %w", err)
	}
	if average.Valid {
		summary.AverageRecoveryMins = round1(average.Float64)
	}
	summary.ResponseSLAHitRate = round1(summary.ResponseSLAHitRate)
	summary.RecoverySLAHitRate = round1(summary.RecoverySLAHitRate)
	return summary, nil
}

func (r *Repository) findSite(ctx context.Context, siteID string) (siteTarget, error) {
	var site siteTarget
	err := r.database.Pool().QueryRow(ctx, `
		SELECT id::text, code
		FROM sites
		WHERE deleted_at IS NULL AND ($1 = '' OR id::text = $1 OR code = $1)
		ORDER BY code
		LIMIT 1
	`, siteID).Scan(&site.ID, &site.Code)
	if errors.Is(err, pgx.ErrNoRows) {
		return siteTarget{}, ErrNotFound
	}
	if err != nil {
		return siteTarget{}, fmt.Errorf("query maintenance site: %w", err)
	}
	return site, nil
}

func (r *Repository) getWorkOrder(ctx context.Context, workOrderID string) (WorkOrder, error) {
	rows, err := r.database.Pool().Query(ctx, workOrderSQL+`
		WHERE wo.deleted_at IS NULL AND (wo.id::text = $1 OR wo.work_order_no = $1)
	`, workOrderID)
	if err != nil {
		return WorkOrder{}, fmt.Errorf("query work order: %w", err)
	}
	defer rows.Close()

	items, err := scanWorkOrders(rows)
	if err != nil {
		return WorkOrder{}, err
	}
	if len(items) == 0 {
		return WorkOrder{}, ErrNotFound
	}
	return items[0], nil
}

type siteTarget struct {
	ID   string
	Code string
}

type faultTarget struct {
	FaultID      string
	FaultNo      string
	SiteID       string
	ChargerID    string
	ChargerCode  string
	ConnectorID  *string
	SessionID    *string
	FaultCode    string
	Severity     string
	WorkOrderID  string
	ResponseMins int
	RecoveryMins int
}

func findFaultTarget(ctx context.Context, tx txExecutor, faultID string) (faultTarget, error) {
	var target faultTarget
	var connectorID, sessionID, workOrderID sql.NullString
	err := tx.QueryRow(ctx, `
		SELECT
			cf.id::text,
			cf.fault_no,
			g.site_id::text,
			c.id::text,
			c.code,
			cf.connector_id::text,
			cf.session_id::text,
			cf.fault_code,
			cf.severity,
			wo.id::text,
			s.sla_response_minutes,
			s.sla_recovery_minutes
		FROM charger_faults cf
		INNER JOIN chargers c ON c.id = cf.charger_id
		INNER JOIN charger_groups g ON g.id = c.group_id
		INNER JOIN sites s ON s.id = g.site_id
		LEFT JOIN work_orders wo ON wo.fault_id = cf.id AND wo.deleted_at IS NULL
		WHERE cf.deleted_at IS NULL AND (cf.id::text = $1 OR cf.fault_no = $1)
	`, faultID).Scan(
		&target.FaultID,
		&target.FaultNo,
		&target.SiteID,
		&target.ChargerID,
		&target.ChargerCode,
		&connectorID,
		&sessionID,
		&target.FaultCode,
		&target.Severity,
		&workOrderID,
		&target.ResponseMins,
		&target.RecoveryMins,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return faultTarget{}, ErrNotFound
	}
	if err != nil {
		return faultTarget{}, fmt.Errorf("query fault target: %w", err)
	}
	target.ConnectorID = nullableString(connectorID)
	target.SessionID = nullableString(sessionID)
	target.WorkOrderID = workOrderID.String
	return target, nil
}

func insertWorkOrder(ctx context.Context, tx txExecutor, target faultTarget, params CreateWorkOrderParams) (string, error) {
	now := time.Now().UTC()
	workOrderNo := "WO-" + now.Format("20060102150405.000000000")
	status := targetInitialStatus(params)
	title := params.Title
	if title == "" {
		title = target.ChargerCode + " / " + target.FaultCode
	}
	var workOrderID string
	err := tx.QueryRow(ctx, `
		INSERT INTO work_orders (
			work_order_no, fault_id, site_id, charger_id, connector_id, session_id,
			severity, status, impact_scope, title, description, assignee_name,
			response_due_at, recovery_due_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING id::text
	`,
		workOrderNo,
		target.FaultID,
		target.SiteID,
		target.ChargerID,
		target.ConnectorID,
		target.SessionID,
		target.Severity,
		status,
		params.ImpactScope,
		title,
		params.Description,
		params.AssigneeName,
		now.Add(time.Duration(target.ResponseMins)*time.Minute),
		now.Add(time.Duration(target.RecoveryMins)*time.Minute),
	).Scan(&workOrderID)
	if err != nil {
		return "", fmt.Errorf("insert work order: %w", err)
	}
	return workOrderID, nil
}

func targetInitialStatus(params CreateWorkOrderParams) string {
	if params.AssigneeName != "" {
		return "assigned"
	}
	return "open"
}

func currentWorkOrderStatus(ctx context.Context, tx txExecutor, workOrderID string) (string, error) {
	var status string
	err := tx.QueryRow(ctx, `
		SELECT status
		FROM work_orders
		WHERE deleted_at IS NULL AND (id::text = $1 OR work_order_no = $1)
	`, workOrderID).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("query work-order status: %w", err)
	}
	return status, nil
}

func updateWorkOrderStatus(ctx context.Context, tx txExecutor, params TransitionParams, current string) error {
	_, err := tx.Exec(ctx, `
		UPDATE work_orders
		SET
			status = $2,
			assignee_name = CASE WHEN $3 = '' THEN assignee_name ELSE $3 END,
			accepted_at = CASE WHEN $2 = 'accepted' THEN COALESCE(accepted_at, now()) ELSE accepted_at END,
			arrived_at = CASE WHEN $2 = 'arrived' THEN COALESCE(arrived_at, now()) ELSE arrived_at END,
			handling_at = CASE WHEN $2 = 'handling' THEN COALESCE(handling_at, now()) ELSE handling_at END,
			retest_at = CASE WHEN $2 = 'retest' THEN COALESCE(retest_at, now()) ELSE retest_at END,
			recovered_at = CASE WHEN $2 = 'recovered' THEN COALESCE(recovered_at, now()) ELSE recovered_at END,
			closed_at = CASE WHEN $2 = 'closed' THEN COALESCE(closed_at, now()) ELSE closed_at END,
			sla_response_breached = CASE
				WHEN $2 IN ('accepted', 'arrived', 'handling', 'retest', 'recovered', 'closed') THEN now() > response_due_at
				ELSE sla_response_breached
			END,
			sla_recovery_breached = CASE
				WHEN $2 IN ('recovered', 'closed') THEN now() > recovery_due_at
				ELSE sla_recovery_breached
			END
		WHERE deleted_at IS NULL AND (id::text = $1 OR work_order_no = $1)
	`, params.WorkOrderID, params.TargetStatus, params.AssigneeName)
	if err != nil {
		return fmt.Errorf("update work-order status: %w", err)
	}
	return nil
}

func resolveLinkedFault(ctx context.Context, tx txExecutor, workOrderID string) error {
	var faultID, chargerID sql.NullString
	var connectorID sql.NullString
	err := tx.QueryRow(ctx, `
		UPDATE charger_faults cf
		SET status = 'resolved', resolved_at = COALESCE(resolved_at, now())
		FROM work_orders wo
		WHERE wo.fault_id = cf.id
		  AND (wo.id::text = $1 OR wo.work_order_no = $1)
		  AND cf.deleted_at IS NULL
		RETURNING cf.id::text, cf.charger_id::text, cf.connector_id::text
	`, workOrderID).Scan(&faultID, &chargerID, &connectorID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("resolve linked fault: %w", err)
	}
	if chargerID.Valid {
		if _, err := tx.Exec(ctx, "UPDATE chargers SET status = 'available' WHERE id::text = $1 AND status = 'fault'", chargerID.String); err != nil {
			return fmt.Errorf("recover charger: %w", err)
		}
	}
	if connectorID.Valid {
		if _, err := tx.Exec(ctx, "UPDATE connectors SET status = 'available' WHERE id::text = $1 AND status = 'fault'", connectorID.String); err != nil {
			return fmt.Errorf("recover connector: %w", err)
		}
	}
	return nil
}

func insertWorkOrderEvent(
	ctx context.Context,
	tx txExecutor,
	workOrderID string,
	eventType string,
	fromStatus string,
	toStatus string,
	actorName string,
	note string,
	payload json.RawMessage,
) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO work_order_events (
			work_order_id, event_type, from_status, to_status, actor_name, note, payload
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, workOrderID, eventType, fromStatus, toStatus, actorName, note, payload)
	if err != nil {
		return fmt.Errorf("insert work-order event: %w", err)
	}
	return nil
}

func insertAudit(ctx context.Context, tx txExecutor, entityType string, entityID string, action string, actor string, payload json.RawMessage) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO audit_logs (entity_type, entity_id, action, actor_name, payload)
		VALUES ($1, $2, $3, $4, $5)
	`, entityType, nullableStringValue(entityID), action, actor, payload)
	if err != nil {
		return fmt.Errorf("insert audit log: %w", err)
	}
	return nil
}

const faultSQL = `
	SELECT
		cf.id::text,
		cf.fault_no,
		g.site_id::text,
		s.code,
		c.id::text,
		c.code,
		cf.connector_id::text,
		COALESCE(cn.code, ''),
		cf.session_id::text,
		COALESCE(cs.session_no, ''),
		cf.fault_code,
		cf.severity,
		cf.status,
		wo.id::text,
		COALESCE(wo.work_order_no, ''),
		COUNT(cf.id) OVER (PARTITION BY cf.charger_id, cf.fault_code)::int,
		cf.occurred_at,
		cf.resolved_at
	FROM charger_faults cf
	INNER JOIN chargers c ON c.id = cf.charger_id
	INNER JOIN charger_groups g ON g.id = c.group_id
	INNER JOIN sites s ON s.id = g.site_id
	LEFT JOIN connectors cn ON cn.id = cf.connector_id
	LEFT JOIN charging_sessions cs ON cs.id = cf.session_id
	LEFT JOIN work_orders wo ON wo.fault_id = cf.id AND wo.deleted_at IS NULL
`

const workOrderSQL = `
	SELECT
		wo.id::text,
		wo.work_order_no,
		wo.fault_id::text,
		COALESCE(cf.fault_no, ''),
		wo.site_id::text,
		s.code,
		wo.charger_id::text,
		c.code,
		wo.connector_id::text,
		COALESCE(cn.code, ''),
		wo.session_id::text,
		COALESCE(cs.session_no, ''),
		wo.severity,
		wo.status,
		wo.impact_scope,
		wo.title,
		wo.description,
		wo.assignee_name,
		wo.response_due_at,
		wo.recovery_due_at,
		wo.accepted_at,
		wo.arrived_at,
		wo.handling_at,
		wo.retest_at,
		wo.recovered_at,
		wo.closed_at,
		wo.sla_response_breached,
		wo.sla_recovery_breached,
		wo.created_at,
		wo.updated_at
	FROM work_orders wo
	INNER JOIN sites s ON s.id = wo.site_id
	INNER JOIN chargers c ON c.id = wo.charger_id
	LEFT JOIN charger_faults cf ON cf.id = wo.fault_id
	LEFT JOIN connectors cn ON cn.id = wo.connector_id
	LEFT JOIN charging_sessions cs ON cs.id = wo.session_id
`

func scanFaults(rows pgx.Rows) ([]Fault, error) {
	faults := make([]Fault, 0)
	for rows.Next() {
		var item Fault
		var connectorID, sessionID, workOrderID sql.NullString
		var resolvedAt sql.NullTime
		if err := rows.Scan(
			&item.ID,
			&item.FaultNo,
			&item.SiteID,
			&item.SiteCode,
			&item.ChargerID,
			&item.ChargerCode,
			&connectorID,
			&item.ConnectorCode,
			&sessionID,
			&item.SessionNo,
			&item.FaultCode,
			&item.Severity,
			&item.Status,
			&workOrderID,
			&item.WorkOrderNo,
			&item.RepeatCount,
			&item.OccurredAt,
			&resolvedAt,
		); err != nil {
			return nil, fmt.Errorf("scan fault: %w", err)
		}
		item.ConnectorID = nullableString(connectorID)
		item.SessionID = nullableString(sessionID)
		item.WorkOrderID = nullableString(workOrderID)
		item.ResolvedAt = nullableTime(resolvedAt)
		faults = append(faults, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate faults: %w", err)
	}
	return faults, nil
}

func scanWorkOrders(rows pgx.Rows) ([]WorkOrder, error) {
	items := make([]WorkOrder, 0)
	for rows.Next() {
		item, err := scanWorkOrder(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate work orders: %w", err)
	}
	return items, nil
}

func scanWorkOrder(scanner interface{ Scan(dest ...any) error }) (WorkOrder, error) {
	var item WorkOrder
	var faultID, connectorID, sessionID sql.NullString
	var acceptedAt, arrivedAt, handlingAt, retestAt, recoveredAt, closedAt sql.NullTime
	if err := scanner.Scan(
		&item.ID,
		&item.WorkOrderNo,
		&faultID,
		&item.FaultNo,
		&item.SiteID,
		&item.SiteCode,
		&item.ChargerID,
		&item.ChargerCode,
		&connectorID,
		&item.ConnectorCode,
		&sessionID,
		&item.SessionNo,
		&item.Severity,
		&item.Status,
		&item.ImpactScope,
		&item.Title,
		&item.Description,
		&item.AssigneeName,
		&item.ResponseDueAt,
		&item.RecoveryDueAt,
		&acceptedAt,
		&arrivedAt,
		&handlingAt,
		&retestAt,
		&recoveredAt,
		&closedAt,
		&item.SLAResponseBreached,
		&item.SLARecoveryBreached,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return WorkOrder{}, fmt.Errorf("scan work order: %w", err)
	}
	item.FaultID = nullableString(faultID)
	item.ConnectorID = nullableString(connectorID)
	item.SessionID = nullableString(sessionID)
	item.AcceptedAt = nullableTime(acceptedAt)
	item.ArrivedAt = nullableTime(arrivedAt)
	item.HandlingAt = nullableTime(handlingAt)
	item.RetestAt = nullableTime(retestAt)
	item.RecoveredAt = nullableTime(recoveredAt)
	item.ClosedAt = nullableTime(closedAt)
	return item, nil
}

type txExecutor interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
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

func round1(value float64) float64 {
	return math.Round(value*10) / 10
}
