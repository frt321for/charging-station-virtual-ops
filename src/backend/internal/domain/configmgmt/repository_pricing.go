package configmgmt

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5"
)

func (r *Repository) listPricingPolicies(ctx context.Context, siteID string) ([]PricingPolicy, error) {
	rows, err := r.database.Pool().Query(ctx, pricingPolicySQL+`
		WHERE pp.deleted_at IS NULL AND pp.site_id::text = $1
		ORDER BY pp.code, pp.version DESC, pr.start_minute
	`, siteID)
	if err != nil {
		return nil, fmt.Errorf("query pricing config: %w", err)
	}
	defer rows.Close()
	return scanPricingPolicies(rows)
}

// CreatePricingPolicy creates a new pricing policy version.
func (r *Repository) CreatePricingPolicy(ctx context.Context, params PricingPolicyCreate) (PricingPolicy, error) {
	site, err := r.FindSiteBoundary(ctx, params.SiteID)
	if err != nil {
		return PricingPolicy{}, err
	}
	tx, err := r.database.Pool().Begin(ctx)
	if err != nil {
		return PricingPolicy{}, fmt.Errorf("begin pricing config: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var version int
	if err := tx.QueryRow(ctx, `
		SELECT COALESCE(MAX(version), 0) + 1
		FROM pricing_policies
		WHERE site_id::text = $1 AND code = $2 AND deleted_at IS NULL
	`, site.SiteID, params.Code).Scan(&version); err != nil {
		return PricingPolicy{}, fmt.Errorf("query next pricing version: %w", err)
	}

	var policyID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO pricing_policies (
			site_id, code, name, version, charger_type, effective_from, status
		)
		VALUES ($1, $2, $3, $4, $5, now(), $6)
		RETURNING id::text
	`, site.SiteID, params.Code, params.Name, version, params.ChargerType, params.Status).Scan(&policyID); err != nil {
		return PricingPolicy{}, fmt.Errorf("insert pricing policy: %w", err)
	}

	for _, period := range params.Periods {
		if _, err := tx.Exec(ctx, `
			INSERT INTO pricing_periods (
				policy_id, label, start_minute, end_minute, energy_price_per_kwh,
				service_fee_per_kwh, occupancy_fee_per_minute
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, policyID, period.Label, period.StartMinute, period.EndMinute, period.EnergyPricePerKWh, period.ServiceFeePerKWh, period.OccupancyFeePerMinute); err != nil {
			return PricingPolicy{}, fmt.Errorf("insert pricing period: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return PricingPolicy{}, fmt.Errorf("commit pricing config: %w", err)
	}
	if err := r.audit(ctx, "pricing_policy", policyID, "config.create_pricing_policy", params); err != nil {
		return PricingPolicy{}, err
	}
	return r.getPricingPolicy(ctx, policyID)
}

func (r *Repository) getPricingPolicy(ctx context.Context, policyID string) (PricingPolicy, error) {
	rows, err := r.database.Pool().Query(ctx, pricingPolicySQL+`
		WHERE pp.id::text = $1
		ORDER BY pr.start_minute
	`, policyID)
	if err != nil {
		return PricingPolicy{}, fmt.Errorf("query created pricing policy: %w", err)
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
		return nil, fmt.Errorf("iterate pricing config: %w", err)
	}
	return policies, nil
}

func scanPricingPolicyRow(rows pgx.Rows) (PricingPolicy, *PricingPeriod, error) {
	var policy PricingPolicy
	var effectiveTo sql.NullTime
	var periodID, label sql.NullString
	var startMinute, endMinute sql.NullInt64
	var energyPrice, serviceFee, occupancyFee sql.NullFloat64
	err := rows.Scan(
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
		return PricingPolicy{}, nil, fmt.Errorf("scan pricing config: %w", err)
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
