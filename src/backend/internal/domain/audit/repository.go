package audit

import (
	"context"
	"database/sql"
	"fmt"

	"charging-ops/backend/internal/platform/database"
)

// Repository reads audit log rows from PostgreSQL.
type Repository struct {
	database *database.Client
}

// NewRepository creates an audit repository.
func NewRepository(database *database.Client) *Repository {
	return &Repository{database: database}
}

// List returns permission-sensitive audit logs.
func (r *Repository) List(ctx context.Context, params ListParams) (Page, error) {
	params = normalizeParams(params)
	total, err := r.count(ctx, params)
	if err != nil {
		return Page{}, err
	}

	rows, err := r.database.Pool().Query(ctx, `
		SELECT
			id::text,
			entity_type,
			entity_id::text,
			action,
			actor_user_id::text,
			actor_name,
			actor_role_code,
			ip_address,
			trace_id,
			payload,
			created_at
		FROM audit_logs
		WHERE ($1 = '' OR entity_type ILIKE '%' || $1 || '%' OR entity_id::text ILIKE '%' || $1 || '%')
		  AND ($2 = '' OR action = $2)
		  AND ($3 = '' OR actor_name ILIKE '%' || $3 || '%')
		ORDER BY created_at DESC, id DESC
		LIMIT $4 OFFSET $5
	`, params.EntityType, params.Action, params.ActorName, params.PageSize, offset(params))
	if err != nil {
		return Page{}, fmt.Errorf("query audit logs: %w", err)
	}
	defer rows.Close()

	logs := make([]Log, 0)
	for rows.Next() {
		item, err := scanLog(rows)
		if err != nil {
			return Page{}, err
		}
		logs = append(logs, item)
	}
	if err := rows.Err(); err != nil {
		return Page{}, fmt.Errorf("iterate audit logs: %w", err)
	}

	return Page{
		List: logs,
		Pagination: Pagination{
			Page:       params.Page,
			PageSize:   params.PageSize,
			Total:      total,
			TotalPages: totalPages(total, params.PageSize),
		},
	}, nil
}

func (r *Repository) count(ctx context.Context, params ListParams) (int, error) {
	var total int
	err := r.database.Pool().QueryRow(ctx, `
		SELECT COUNT(*)::int
		FROM audit_logs
		WHERE ($1 = '' OR entity_type ILIKE '%' || $1 || '%' OR entity_id::text ILIKE '%' || $1 || '%')
		  AND ($2 = '' OR action = $2)
		  AND ($3 = '' OR actor_name ILIKE '%' || $3 || '%')
	`, params.EntityType, params.Action, params.ActorName).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("count audit logs: %w", err)
	}
	return total, nil
}

func scanLog(scanner interface{ Scan(dest ...any) error }) (Log, error) {
	var item Log
	var entityID, actorUserID sql.NullString
	var payload []byte
	if err := scanner.Scan(
		&item.ID,
		&item.EntityType,
		&entityID,
		&item.Action,
		&actorUserID,
		&item.ActorName,
		&item.ActorRoleCode,
		&item.IPAddress,
		&item.TraceID,
		&payload,
		&item.CreatedAt,
	); err != nil {
		return Log{}, fmt.Errorf("scan audit log: %w", err)
	}
	item.EntityID = nullableString(entityID)
	item.ActorUserID = nullableString(actorUserID)
	item.Payload = normalizeJSON(payload)
	return item, nil
}

func normalizeParams(params ListParams) ListParams {
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.PageSize <= 0 {
		params.PageSize = 20
	}
	if params.PageSize > 100 {
		params.PageSize = 100
	}
	return params
}

func offset(params ListParams) int {
	return (params.Page - 1) * params.PageSize
}

func totalPages(total int, pageSize int) int {
	if total == 0 {
		return 0
	}
	return (total + pageSize - 1) / pageSize
}

func nullableString(value sql.NullString) *string {
	if !value.Valid || value.String == "" {
		return nil
	}
	return &value.String
}

func normalizeJSON(payload []byte) []byte {
	if len(payload) == 0 {
		return []byte(`{}`)
	}
	return payload
}
