package billing

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"charging-ops/backend/internal/platform/database"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Repository persists pricing, billing drafts, and reconciliation facts.
type Repository struct {
	database *database.Client
}

// NewRepository creates a billing repository.
func NewRepository(database *database.Client) *Repository {
	return &Repository{database: database}
}

// ListPolicies returns pricing policies with their periods.
func (r *Repository) ListPolicies(ctx context.Context, siteID string) ([]PricingPolicy, error) {
	rows, err := r.database.Pool().Query(ctx, pricingPolicySQL+`
		WHERE pp.deleted_at IS NULL
		  AND ($1 = '' OR s.id::text = $1 OR s.code = $1)
		ORDER BY s.code, pp.code, pp.version DESC, pr.start_minute
	`, siteID)
	if err != nil {
		return nil, fmt.Errorf("query pricing policies: %w", err)
	}
	defer rows.Close()
	return scanPricingPolicies(rows)
}

// FindActivePolicy resolves the active pricing policy for a session.
func (r *Repository) FindActivePolicy(ctx context.Context, siteID string, chargerType string) (PricingPolicy, error) {
	rows, err := r.database.Pool().Query(ctx, pricingPolicySQL+`
		WHERE pp.deleted_at IS NULL
		  AND pp.site_id::text = $1
		  AND pp.status = 'active'
		  AND pp.effective_from <= now()
		  AND (pp.effective_to IS NULL OR pp.effective_to > now())
		  AND pp.charger_type IN ('all', $2)
		ORDER BY
		  CASE WHEN pp.charger_type = $2 THEN 0 ELSE 1 END,
		  pp.version DESC,
		  pr.start_minute
		LIMIT 24
	`, siteID, chargerType)
	if err != nil {
		return PricingPolicy{}, fmt.Errorf("query active pricing policy: %w", err)
	}
	defer rows.Close()

	policies, err := scanPricingPolicies(rows)
	if err != nil {
		return PricingPolicy{}, err
	}
	if len(policies) == 0 {
		return PricingPolicy{}, ErrNotFound
	}
	return policies[0], nil
}

// ListDrafts returns recent billing drafts.
func (r *Repository) ListDrafts(ctx context.Context, siteID string) ([]BillingDraft, error) {
	rows, err := r.database.Pool().Query(ctx, billingDraftSQL+`
		WHERE bd.deleted_at IS NULL
		  AND ($1 = '' OR s.id::text = $1 OR s.code = $1)
		ORDER BY bd.generated_at DESC
		LIMIT 50
	`, siteID)
	if err != nil {
		return nil, fmt.Errorf("query billing drafts: %w", err)
	}
	defer rows.Close()
	return scanBillingDrafts(rows)
}

// ListExceptions returns open reconciliation exceptions.
func (r *Repository) ListExceptions(ctx context.Context, siteID string) ([]ReconciliationException, error) {
	rows, err := r.database.Pool().Query(ctx, reconciliationExceptionSQL+`
		WHERE re.deleted_at IS NULL
		  AND re.status <> 'resolved'
		  AND ($1 = '' OR s.id::text = $1 OR s.code = $1)
		ORDER BY re.detected_at DESC
		LIMIT 50
	`, siteID)
	if err != nil {
		return nil, fmt.Errorf("query reconciliation exceptions: %w", err)
	}
	defer rows.Close()
	return scanExceptions(rows)
}

// FindSessionForDraft loads all data needed to generate a bill.
func (r *Repository) FindSessionForDraft(ctx context.Context, sessionID string) (sessionForDraft, error) {
	var item sessionForDraft
	var startedAt, stoppedAt sql.NullTime
	var meterStart, meterStop, firstMeter, lastMeter sql.NullFloat64
	err := r.database.Pool().QueryRow(ctx, `
		WITH meter_bounds AS (
			SELECT
				session_id,
				(array_agg(meter_kwh ORDER BY time ASC))[1]::float8 AS first_meter_kwh,
				(array_agg(meter_kwh ORDER BY time DESC))[1]::float8 AS last_meter_kwh,
				COUNT(*)::int AS sample_count
			FROM charger_meter_values
			WHERE session_id IS NOT NULL
			GROUP BY session_id
		)
		SELECT
			cs.id::text,
			cs.session_no,
			s.id::text,
			s.code,
			cn.code,
			c.charger_type,
			cs.status,
			cs.started_at,
			cs.stopped_at,
			cs.stop_reason,
			cs.meter_start_kwh,
			cs.meter_stop_kwh,
			mb.first_meter_kwh,
			mb.last_meter_kwh,
			COALESCE(mb.sample_count, 0)
		FROM charging_sessions cs
		INNER JOIN sites s ON s.id = cs.site_id
		INNER JOIN chargers c ON c.id = cs.charger_id
		INNER JOIN connectors cn ON cn.id = cs.connector_id
		LEFT JOIN meter_bounds mb ON mb.session_id = cs.id
		WHERE cs.deleted_at IS NULL AND (cs.id::text = $1 OR cs.session_no = $1)
	`, sessionID).Scan(
		&item.SessionID,
		&item.SessionNo,
		&item.SiteID,
		&item.SiteCode,
		&item.ConnectorCode,
		&item.ChargerType,
		&item.Status,
		&startedAt,
		&stoppedAt,
		&item.StopReason,
		&meterStart,
		&meterStop,
		&firstMeter,
		&lastMeter,
		&item.MeterSampleCount,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return sessionForDraft{}, ErrNotFound
	}
	if err != nil {
		return sessionForDraft{}, fmt.Errorf("query session for billing draft: %w", err)
	}

	item.StartedAt = nullableTime(startedAt)
	item.StoppedAt = nullableTime(stoppedAt)
	item.MeterStartKWh = nullableFloat(meterStart)
	item.MeterStopKWh = nullableFloat(meterStop)
	item.FirstMeterKWh = nullableFloat(firstMeter)
	item.LastMeterKWh = nullableFloat(lastMeter)
	return item, nil
}

// SaveDraft upserts a billing draft and replaces open exceptions for it.
func (r *Repository) SaveDraft(ctx context.Context, calculation draftCalculation, generatedBy string) (BillingDraft, error) {
	tx, err := r.database.Pool().Begin(ctx)
	if err != nil {
		return BillingDraft{}, fmt.Errorf("begin billing draft: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	billNo := "BILL-" + time.Now().UTC().Format("20060102150405.000000000")
	status := "draft"
	if calculation.ExceptionFlag {
		status = "pending_review"
	}

	var billID string
	err = tx.QueryRow(ctx, `
		INSERT INTO billing_drafts (
			bill_no, session_id, pricing_policy_id, energy_kwh, duration_minutes,
			energy_amount, service_amount, occupancy_amount, total_amount,
			status, exception_flag, generated_by, generated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, now())
		ON CONFLICT (session_id) DO UPDATE SET
			pricing_policy_id = EXCLUDED.pricing_policy_id,
			energy_kwh = EXCLUDED.energy_kwh,
			duration_minutes = EXCLUDED.duration_minutes,
			energy_amount = EXCLUDED.energy_amount,
			service_amount = EXCLUDED.service_amount,
			occupancy_amount = EXCLUDED.occupancy_amount,
			total_amount = EXCLUDED.total_amount,
			status = EXCLUDED.status,
			exception_flag = EXCLUDED.exception_flag,
			generated_by = EXCLUDED.generated_by,
			generated_at = now()
		RETURNING id::text
	`,
		billNo,
		calculation.Session.SessionID,
		calculation.Policy.ID,
		calculation.EnergyKWh,
		calculation.DurationMinutes,
		calculation.EnergyAmount,
		calculation.ServiceAmount,
		calculation.OccupancyAmount,
		calculation.TotalAmount,
		status,
		calculation.ExceptionFlag,
		generatedBy,
	).Scan(&billID)
	if err != nil {
		return BillingDraft{}, fmt.Errorf("upsert billing draft: %w", err)
	}

	if err := replaceExceptions(ctx, tx, billID, calculation); err != nil {
		return BillingDraft{}, err
	}
	if err := insertBillingEvent(ctx, tx, billID, calculation); err != nil {
		return BillingDraft{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return BillingDraft{}, fmt.Errorf("commit billing draft: %w", err)
	}
	return r.getDraft(ctx, billID)
}

func (r *Repository) getDraft(ctx context.Context, billID string) (BillingDraft, error) {
	rows, err := r.database.Pool().Query(ctx, billingDraftSQL+`
		WHERE bd.id::text = $1
	`, billID)
	if err != nil {
		return BillingDraft{}, fmt.Errorf("query billing draft: %w", err)
	}
	defer rows.Close()

	drafts, err := scanBillingDrafts(rows)
	if err != nil {
		return BillingDraft{}, err
	}
	if len(drafts) == 0 {
		return BillingDraft{}, ErrNotFound
	}
	return drafts[0], nil
}

const pricingPolicySQL = `
	SELECT
		pp.id::text,
		pp.site_id::text,
		s.code,
		pp.code,
		pp.name,
		pp.version,
		pp.charger_type,
		pp.effective_from,
		pp.effective_to,
		pp.status,
		pr.id::text,
		pr.label,
		pr.start_minute,
		pr.end_minute,
		pr.energy_price_per_kwh::float8,
		pr.service_fee_per_kwh::float8,
		pr.occupancy_fee_per_minute::float8
	FROM pricing_policies pp
	INNER JOIN sites s ON s.id = pp.site_id
	LEFT JOIN pricing_periods pr ON pr.policy_id = pp.id
`

const billingDraftSQL = `
	SELECT
		bd.id::text,
		bd.bill_no,
		bd.session_id::text,
		cs.session_no,
		s.id::text,
		s.code,
		cn.code,
		bd.pricing_policy_id::text,
		pp.code,
		pp.version,
		bd.energy_kwh::float8,
		bd.duration_minutes,
		bd.energy_amount::float8,
		bd.service_amount::float8,
		bd.occupancy_amount::float8,
		bd.total_amount::float8,
		bd.currency,
		bd.status,
		bd.exception_flag,
		bd.generated_by,
		bd.generated_at
	FROM billing_drafts bd
	INNER JOIN charging_sessions cs ON cs.id = bd.session_id
	INNER JOIN sites s ON s.id = cs.site_id
	INNER JOIN connectors cn ON cn.id = cs.connector_id
	INNER JOIN pricing_policies pp ON pp.id = bd.pricing_policy_id
`

const reconciliationExceptionSQL = `
	SELECT
		re.id::text,
		re.exception_no,
		re.bill_id::text,
		bd.bill_no,
		re.session_id::text,
		cs.session_no,
		re.exception_type,
		re.severity,
		re.status,
		re.reason,
		re.suggested_action,
		re.detected_at,
		re.resolved_at
	FROM reconciliation_exceptions re
	INNER JOIN billing_drafts bd ON bd.id = re.bill_id
	INNER JOIN charging_sessions cs ON cs.id = re.session_id
	INNER JOIN sites s ON s.id = cs.site_id
`

type pricingPolicyScanner interface {
	Scan(dest ...any) error
}

func scanPricingPolicies(rows pgx.Rows) ([]PricingPolicy, error) {
	policies := make([]PricingPolicy, 0)
	indexes := make(map[string]int)
	for rows.Next() {
		policy, period, err := scanPricingPolicyRow(rows)
		if err != nil {
			return nil, err
		}
		index, ok := indexes[policy.ID]
		if !ok {
			policies = append(policies, policy)
			index = len(policies) - 1
			indexes[policy.ID] = index
		}
		if period != nil {
			policies[index].Periods = append(policies[index].Periods, *period)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate pricing policies: %w", err)
	}
	return policies, nil
}

func scanPricingPolicyRow(scanner pricingPolicyScanner) (PricingPolicy, *PricingPeriod, error) {
	var policy PricingPolicy
	var effectiveTo sql.NullTime
	var periodID, label sql.NullString
	var startMinute, endMinute sql.NullInt64
	var energyPrice, serviceFee, occupancyFee sql.NullFloat64
	err := scanner.Scan(
		&policy.ID,
		&policy.SiteID,
		&policy.SiteCode,
		&policy.Code,
		&policy.Name,
		&policy.Version,
		&policy.ChargerType,
		&policy.EffectiveFrom,
		&effectiveTo,
		&policy.Status,
		&periodID,
		&label,
		&startMinute,
		&endMinute,
		&energyPrice,
		&serviceFee,
		&occupancyFee,
	)
	if err != nil {
		return PricingPolicy{}, nil, fmt.Errorf("scan pricing policy: %w", err)
	}
	policy.EffectiveTo = nullableTime(effectiveTo)
	if !periodID.Valid {
		return policy, nil, nil
	}
	return policy, &PricingPeriod{
		ID:                    periodID.String,
		Label:                 label.String,
		StartMinute:           int(startMinute.Int64),
		EndMinute:             int(endMinute.Int64),
		EnergyPricePerKWh:     energyPrice.Float64,
		ServiceFeePerKWh:      serviceFee.Float64,
		OccupancyFeePerMinute: occupancyFee.Float64,
	}, nil
}

func scanBillingDrafts(rows pgx.Rows) ([]BillingDraft, error) {
	drafts := make([]BillingDraft, 0)
	for rows.Next() {
		var draft BillingDraft
		if err := rows.Scan(
			&draft.ID,
			&draft.BillNo,
			&draft.SessionID,
			&draft.SessionNo,
			&draft.SiteID,
			&draft.SiteCode,
			&draft.ConnectorCode,
			&draft.PricingPolicyID,
			&draft.PolicyCode,
			&draft.PolicyVersion,
			&draft.EnergyKWh,
			&draft.DurationMinutes,
			&draft.EnergyAmount,
			&draft.ServiceAmount,
			&draft.OccupancyAmount,
			&draft.TotalAmount,
			&draft.Currency,
			&draft.Status,
			&draft.ExceptionFlag,
			&draft.GeneratedBy,
			&draft.GeneratedAt,
		); err != nil {
			return nil, fmt.Errorf("scan billing draft: %w", err)
		}
		drafts = append(drafts, draft)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate billing drafts: %w", err)
	}
	return drafts, nil
}

func scanExceptions(rows pgx.Rows) ([]ReconciliationException, error) {
	exceptions := make([]ReconciliationException, 0)
	for rows.Next() {
		var item ReconciliationException
		var resolvedAt sql.NullTime
		if err := rows.Scan(
			&item.ID,
			&item.ExceptionNo,
			&item.BillID,
			&item.BillNo,
			&item.SessionID,
			&item.SessionNo,
			&item.ExceptionType,
			&item.Severity,
			&item.Status,
			&item.Reason,
			&item.SuggestedAction,
			&item.DetectedAt,
			&resolvedAt,
		); err != nil {
			return nil, fmt.Errorf("scan reconciliation exception: %w", err)
		}
		item.ResolvedAt = nullableTime(resolvedAt)
		exceptions = append(exceptions, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate reconciliation exceptions: %w", err)
	}
	return exceptions, nil
}

type txExecutor interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

func replaceExceptions(ctx context.Context, tx txExecutor, billID string, calculation draftCalculation) error {
	if _, err := tx.Exec(ctx, `
		UPDATE reconciliation_exceptions
		SET status = 'resolved', resolved_at = now()
		WHERE bill_id::text = $1 AND status <> 'resolved' AND deleted_at IS NULL
	`, billID); err != nil {
		return fmt.Errorf("resolve previous exceptions: %w", err)
	}

	for index, item := range calculation.Exceptions {
		exceptionNo := fmt.Sprintf("RE-%s-%02d", time.Now().UTC().Format("20060102150405.000000000"), index+1)
		if _, err := tx.Exec(ctx, `
			INSERT INTO reconciliation_exceptions (
				exception_no, bill_id, session_id, exception_type, severity,
				status, reason, suggested_action
			)
			VALUES ($1, $2, $3, $4, $5, 'open', $6, $7)
		`,
			exceptionNo,
			billID,
			calculation.Session.SessionID,
			item.Type,
			item.Severity,
			item.Reason,
			item.SuggestedAction,
		); err != nil {
			return fmt.Errorf("insert reconciliation exception: %w", err)
		}
	}
	return nil
}

func insertBillingEvent(ctx context.Context, tx txExecutor, billID string, calculation draftCalculation) error {
	payload := fmt.Sprintf(
		`{"billId":%q,"energyKwh":%.3f,"totalAmount":%.2f,"exceptionFlag":%t}`,
		billID,
		calculation.EnergyKWh,
		calculation.TotalAmount,
		calculation.ExceptionFlag,
	)
	_, err := tx.Exec(ctx, `
		INSERT INTO session_events (session_id, event_type, source, payload)
		VALUES ($1, 'BillingDraftGenerated', 'billing-engine', $2::jsonb)
	`, calculation.Session.SessionID, payload)
	if err != nil {
		return fmt.Errorf("insert billing event: %w", err)
	}
	return nil
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
