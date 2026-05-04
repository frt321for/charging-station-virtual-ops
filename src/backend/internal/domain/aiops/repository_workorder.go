package aiops

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// FindWorkOrderContext loads a work order with its flow events.
func (r *Repository) FindWorkOrderContext(ctx context.Context, workOrderID string) (workOrderContext, error) {
	data, err := r.findWorkOrderFact(ctx, workOrderID)
	if err != nil {
		return workOrderContext{}, err
	}
	events, err := r.listWorkOrderEvents(ctx, data.WorkOrder.ID)
	if err != nil {
		return workOrderContext{}, err
	}
	data.Events = events
	return data, nil
}

func (r *Repository) findWorkOrderFact(ctx context.Context, workOrderID string) (workOrderContext, error) {
	var data workOrderContext
	var connectorCode, sessionNo sql.NullString
	err := r.database.Pool().QueryRow(ctx, `
		SELECT
			wo.id::text,
			wo.work_order_no,
			wo.status,
			wo.severity,
			wo.title,
			wo.description,
			wo.assignee_name,
			c.code,
			cn.code,
			cs.session_no,
			wo.response_due_at,
			wo.recovery_due_at,
			wo.updated_at,
			s.id::text,
			s.code,
			s.name
		FROM work_orders wo
		INNER JOIN sites s ON s.id = wo.site_id
		INNER JOIN chargers c ON c.id = wo.charger_id
		LEFT JOIN connectors cn ON cn.id = wo.connector_id
		LEFT JOIN charging_sessions cs ON cs.id = wo.session_id
		WHERE wo.deleted_at IS NULL AND (wo.id::text = $1 OR wo.work_order_no = $1)
	`, workOrderID).Scan(
		&data.WorkOrder.ID,
		&data.WorkOrder.WorkOrderNo,
		&data.WorkOrder.Status,
		&data.WorkOrder.Severity,
		&data.WorkOrder.Title,
		&data.WorkOrder.Description,
		&data.WorkOrder.AssigneeName,
		&data.WorkOrder.ChargerCode,
		&connectorCode,
		&sessionNo,
		&data.WorkOrder.ResponseDueAt,
		&data.WorkOrder.RecoveryDueAt,
		&data.WorkOrder.UpdatedAt,
		&data.Site.ID,
		&data.Site.Code,
		&data.Site.Name,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return workOrderContext{}, ErrNotFound
	}
	if err != nil {
		return workOrderContext{}, fmt.Errorf("query ai work order context: %w", err)
	}
	data.WorkOrder.ConnectorCode = scanNullString(connectorCode)
	data.WorkOrder.SessionNo = scanNullString(sessionNo)
	return data, nil
}

func (r *Repository) listWorkOrderEvents(ctx context.Context, workOrderID string) ([]eventFact, error) {
	rows, err := r.database.Pool().Query(ctx, `
		SELECT event_type, actor_name, occurred_at, note
		FROM work_order_events
		WHERE work_order_id::text = $1
		ORDER BY occurred_at DESC, id DESC
		LIMIT 16
	`, workOrderID)
	if err != nil {
		return nil, fmt.Errorf("query ai work-order events: %w", err)
	}
	defer rows.Close()

	items := make([]eventFact, 0)
	for rows.Next() {
		var item eventFact
		if err := rows.Scan(&item.EventType, &item.Source, &item.OccurredAt, &item.Note); err != nil {
			return nil, fmt.Errorf("scan ai work-order event: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate ai work-order events: %w", err)
	}
	return items, nil
}
