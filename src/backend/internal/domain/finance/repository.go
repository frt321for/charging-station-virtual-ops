package finance

import (
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"charging-ops/backend/internal/platform/database"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Repository persists finance review, correction, export, and audit facts.
type Repository struct {
	database *database.Client
}

// NewRepository creates a finance repository.
func NewRepository(database *database.Client) *Repository {
	return &Repository{database: database}
}

// ReviewException updates one reconciliation exception.
func (r *Repository) ReviewException(ctx context.Context, params ReviewParams) error {
	tx, err := r.database.Pool().Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin reconciliation review: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	exceptionID, err := updateExceptionStatus(ctx, tx, params)
	if err != nil {
		return err
	}
	payload := json.RawMessage(fmt.Sprintf(`{"status":%q,"note":%q}`, params.Status, params.Note))
	if err := insertAudit(ctx, tx, "reconciliation_exception", exceptionID, "review", params.ReviewerName, payload); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit reconciliation review: %w", err)
	}
	return nil
}

// CreateCorrection stores a correction record.
func (r *Repository) CreateCorrection(ctx context.Context, params CorrectionParams) (Correction, error) {
	tx, err := r.database.Pool().Begin(ctx)
	if err != nil {
		return Correction{}, fmt.Errorf("begin correction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	target, err := findCorrectionTarget(ctx, tx, params.ExceptionID)
	if err != nil {
		return Correction{}, err
	}
	correctionNo := "BC-" + time.Now().UTC().Format("20060102150405.000000000")
	var correctionID string
	err = tx.QueryRow(ctx, `
		INSERT INTO billing_corrections (
			correction_no, bill_id, exception_id, corrected_energy_kwh,
			corrected_duration_minutes, corrected_total_amount, reason, reviewer_name
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id::text
	`,
		correctionNo,
		target.BillID,
		target.ExceptionID,
		params.CorrectedEnergyKWh,
		params.CorrectedDurationMinutes,
		params.CorrectedTotalAmount,
		params.Reason,
		params.ReviewerName,
	).Scan(&correctionID)
	if err != nil {
		return Correction{}, fmt.Errorf("insert correction: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE reconciliation_exceptions
		SET status = 'reviewing'
		WHERE id::text = $1 AND deleted_at IS NULL
	`, target.ExceptionID); err != nil {
		return Correction{}, fmt.Errorf("mark exception reviewing: %w", err)
	}
	payload := json.RawMessage(fmt.Sprintf(`{"correctionNo":%q,"reason":%q}`, correctionNo, params.Reason))
	if err := insertAudit(ctx, tx, "billing_draft", target.BillID, "correction", params.ReviewerName, payload); err != nil {
		return Correction{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Correction{}, fmt.Errorf("commit correction: %w", err)
	}
	return r.getCorrection(ctx, correctionID)
}

// ConfirmBill confirms a bill and closes linked exceptions.
func (r *Repository) ConfirmBill(ctx context.Context, params ConfirmParams) error {
	tx, err := r.database.Pool().Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin bill confirm: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	target, err := confirmBill(ctx, tx, params)
	if err != nil {
		return err
	}
	payload := json.RawMessage(fmt.Sprintf(`{"billNo":%q,"note":%q}`, target.BillNo, params.Note))
	if err := insertSessionEvent(ctx, tx, target.SessionID, payload); err != nil {
		return err
	}
	if err := insertAudit(ctx, tx, "billing_draft", target.BillID, "confirm", params.ReviewerName, payload); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit bill confirm: %w", err)
	}
	return nil
}

// ExportReconciliation returns export rows and records an audit log.
func (r *Repository) ExportReconciliation(ctx context.Context, siteID string, generatedBy string) (ReconciliationExport, error) {
	rows, err := r.database.Pool().Query(ctx, `
		SELECT
			re.exception_no,
			bd.bill_no,
			cs.session_no,
			re.exception_type,
			re.severity,
			re.status,
			re.reason,
			bd.total_amount::float8
		FROM reconciliation_exceptions re
		INNER JOIN billing_drafts bd ON bd.id = re.bill_id
		INNER JOIN charging_sessions cs ON cs.id = re.session_id
		INNER JOIN sites s ON s.id = cs.site_id
		WHERE re.deleted_at IS NULL
		  AND ($1 = '' OR s.id::text = $1 OR s.code = $1)
		ORDER BY re.detected_at DESC
		LIMIT 200
	`, siteID)
	if err != nil {
		return ReconciliationExport{}, fmt.Errorf("query reconciliation export: %w", err)
	}
	defer rows.Close()

	exportRows := make([]ExportRow, 0)
	for rows.Next() {
		var row ExportRow
		if err := rows.Scan(
			&row.ExceptionNo,
			&row.BillNo,
			&row.SessionNo,
			&row.ExceptionType,
			&row.Severity,
			&row.Status,
			&row.Reason,
			&row.TotalAmount,
		); err != nil {
			return ReconciliationExport{}, fmt.Errorf("scan reconciliation export: %w", err)
		}
		exportRows = append(exportRows, row)
	}
	if err := rows.Err(); err != nil {
		return ReconciliationExport{}, fmt.Errorf("iterate reconciliation export: %w", err)
	}

	export := buildExport(exportRows, generatedBy)
	if err := r.insertExportAudit(ctx, export); err != nil {
		return ReconciliationExport{}, err
	}
	return export, nil
}

func (r *Repository) getCorrection(ctx context.Context, correctionID string) (Correction, error) {
	var correction Correction
	var exceptionID sql.NullString
	var exceptionNo sql.NullString
	var energy, total sql.NullFloat64
	var duration sql.NullInt64
	err := r.database.Pool().QueryRow(ctx, `
		SELECT
			bc.id::text,
			bc.correction_no,
			bc.bill_id::text,
			bd.bill_no,
			bc.exception_id::text,
			re.exception_no,
			bc.corrected_energy_kwh::float8,
			bc.corrected_duration_minutes,
			bc.corrected_total_amount::float8,
			bc.reason,
			bc.reviewer_name,
			bc.status,
			bc.created_at
		FROM billing_corrections bc
		INNER JOIN billing_drafts bd ON bd.id = bc.bill_id
		LEFT JOIN reconciliation_exceptions re ON re.id = bc.exception_id
		WHERE bc.id::text = $1
	`, correctionID).Scan(
		&correction.ID,
		&correction.CorrectionNo,
		&correction.BillID,
		&correction.BillNo,
		&exceptionID,
		&exceptionNo,
		&energy,
		&duration,
		&total,
		&correction.Reason,
		&correction.ReviewerName,
		&correction.Status,
		&correction.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Correction{}, ErrNotFound
	}
	if err != nil {
		return Correction{}, fmt.Errorf("query correction: %w", err)
	}
	correction.ExceptionID = nullableString(exceptionID)
	correction.ExceptionNo = exceptionNo.String
	correction.CorrectedEnergyKWh = nullableFloat(energy)
	correction.CorrectedDurationMinutes = nullableInt(duration)
	correction.CorrectedTotalAmount = nullableFloat(total)
	return correction, nil
}

type correctionTarget struct {
	ExceptionID string
	BillID      string
}

func findCorrectionTarget(ctx context.Context, tx txExecutor, exceptionID string) (correctionTarget, error) {
	var target correctionTarget
	err := tx.QueryRow(ctx, `
		SELECT id::text, bill_id::text
		FROM reconciliation_exceptions
		WHERE deleted_at IS NULL AND (id::text = $1 OR exception_no = $1)
	`, exceptionID).Scan(&target.ExceptionID, &target.BillID)
	if errors.Is(err, pgx.ErrNoRows) {
		return correctionTarget{}, ErrNotFound
	}
	if err != nil {
		return correctionTarget{}, fmt.Errorf("query correction target: %w", err)
	}
	return target, nil
}

func updateExceptionStatus(ctx context.Context, tx txExecutor, params ReviewParams) (string, error) {
	var exceptionID string
	err := tx.QueryRow(ctx, `
		UPDATE reconciliation_exceptions
		SET
			status = $2,
			resolved_at = CASE WHEN $2 = 'resolved' THEN COALESCE(resolved_at, now()) ELSE resolved_at END
		WHERE deleted_at IS NULL AND (id::text = $1 OR exception_no = $1)
		RETURNING id::text
	`, params.ExceptionID, params.Status).Scan(&exceptionID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("update exception review: %w", err)
	}
	return exceptionID, nil
}

type confirmedBill struct {
	BillID    string
	BillNo    string
	SessionID string
}

func confirmBill(ctx context.Context, tx txExecutor, params ConfirmParams) (confirmedBill, error) {
	var target confirmedBill
	err := tx.QueryRow(ctx, `
		UPDATE billing_drafts bd
		SET status = 'confirmed', exception_flag = false
		WHERE bd.deleted_at IS NULL AND (bd.id::text = $1 OR bd.bill_no = $1)
		RETURNING bd.id::text, bd.bill_no, bd.session_id::text
	`, params.BillID).Scan(&target.BillID, &target.BillNo, &target.SessionID)
	if errors.Is(err, pgx.ErrNoRows) {
		return confirmedBill{}, ErrNotFound
	}
	if err != nil {
		return confirmedBill{}, fmt.Errorf("confirm bill: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE reconciliation_exceptions
		SET status = 'resolved', resolved_at = COALESCE(resolved_at, now())
		WHERE bill_id::text = $1 AND deleted_at IS NULL AND status <> 'resolved'
	`, target.BillID); err != nil {
		return confirmedBill{}, fmt.Errorf("resolve bill exceptions: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE charging_sessions
		SET status = 'billed'
		WHERE id::text = $1 AND deleted_at IS NULL
	`, target.SessionID); err != nil {
		return confirmedBill{}, fmt.Errorf("mark session billed: %w", err)
	}
	return target, nil
}

func insertSessionEvent(ctx context.Context, tx txExecutor, sessionID string, payload json.RawMessage) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO session_events (session_id, event_type, source, payload)
		VALUES ($1, 'BillConfirmed', 'finance', $2)
	`, sessionID, payload)
	if err != nil {
		return fmt.Errorf("insert bill confirmation event: %w", err)
	}
	return nil
}

func (r *Repository) insertExportAudit(ctx context.Context, export ReconciliationExport) error {
	payload, err := json.Marshal(map[string]any{
		"exportNo": export.ExportNo,
		"filename": export.Filename,
		"rows":     len(export.Rows),
	})
	if err != nil {
		payload = []byte(`{}`)
	}
	_, err = r.database.Pool().Exec(ctx, `
		INSERT INTO audit_logs (entity_type, action, actor_name, payload)
		VALUES ('reconciliation_exception', 'export', $1, $2)
	`, export.GeneratedBy, payload)
	if err != nil {
		return fmt.Errorf("insert export audit: %w", err)
	}
	return nil
}

func insertAudit(ctx context.Context, tx txExecutor, entityType string, entityID string, action string, actor string, payload json.RawMessage) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO audit_logs (entity_type, entity_id, action, actor_name, payload)
		VALUES ($1, $2, $3, $4, $5)
	`, entityType, nullableStringValue(entityID), action, actor, payload)
	if err != nil {
		return fmt.Errorf("insert finance audit: %w", err)
	}
	return nil
}

func buildExport(rows []ExportRow, generatedBy string) ReconciliationExport {
	now := time.Now().UTC()
	exportNo := "EXP-" + now.Format("20060102150405.000000000")
	var builder strings.Builder
	writer := csv.NewWriter(&builder)
	_ = writer.Write([]string{"exception_no", "bill_no", "session_no", "type", "severity", "status", "reason", "total_amount"})
	for _, row := range rows {
		_ = writer.Write([]string{
			row.ExceptionNo,
			row.BillNo,
			row.SessionNo,
			row.ExceptionType,
			row.Severity,
			row.Status,
			row.Reason,
			strconv.FormatFloat(row.TotalAmount, 'f', 2, 64),
		})
	}
	writer.Flush()
	return ReconciliationExport{
		ExportNo:    exportNo,
		Filename:    exportNo + "-reconciliation.csv",
		GeneratedBy: generatedBy,
		GeneratedAt: now,
		Rows:        rows,
		CSV:         builder.String(),
	}
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

func nullableFloat(value sql.NullFloat64) *float64 {
	if !value.Valid {
		return nil
	}
	return &value.Float64
}

func nullableInt(value sql.NullInt64) *int {
	if !value.Valid {
		return nil
	}
	intValue := int(value.Int64)
	return &intValue
}
