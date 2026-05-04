CREATE TABLE IF NOT EXISTS pricing_policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    site_id UUID NOT NULL REFERENCES sites(id) ON DELETE RESTRICT,
    code TEXT NOT NULL,
    name TEXT NOT NULL,
    version INTEGER NOT NULL CHECK (version > 0),
    charger_type TEXT NOT NULL CHECK (charger_type IN ('all', 'ac', 'dc')),
    effective_from TIMESTAMPTZ NOT NULL,
    effective_to TIMESTAMPTZ NULL,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('draft', 'active', 'retired')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ NULL,
    UNIQUE (site_id, code, version)
);

CREATE INDEX IF NOT EXISTS idx_pricing_policies_site_status ON pricing_policies(site_id, status);

CREATE TRIGGER trg_pricing_policies_updated_at
BEFORE UPDATE ON pricing_policies
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE IF NOT EXISTS pricing_periods (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    policy_id UUID NOT NULL REFERENCES pricing_policies(id) ON DELETE CASCADE,
    label TEXT NOT NULL,
    start_minute INTEGER NOT NULL CHECK (start_minute >= 0 AND start_minute < 1440),
    end_minute INTEGER NOT NULL CHECK (end_minute > 0 AND end_minute <= 1440),
    energy_price_per_kwh NUMERIC(12, 4) NOT NULL CHECK (energy_price_per_kwh >= 0),
    service_fee_per_kwh NUMERIC(12, 4) NOT NULL CHECK (service_fee_per_kwh >= 0),
    occupancy_fee_per_minute NUMERIC(12, 4) NOT NULL DEFAULT 0 CHECK (occupancy_fee_per_minute >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (end_minute > start_minute)
);

CREATE INDEX IF NOT EXISTS idx_pricing_periods_policy ON pricing_periods(policy_id, start_minute);

CREATE TABLE IF NOT EXISTS billing_drafts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bill_no TEXT NOT NULL UNIQUE,
    session_id UUID NOT NULL REFERENCES charging_sessions(id) ON DELETE RESTRICT,
    pricing_policy_id UUID NOT NULL REFERENCES pricing_policies(id) ON DELETE RESTRICT,
    energy_kwh NUMERIC(12, 3) NOT NULL CHECK (energy_kwh >= 0),
    duration_minutes INTEGER NOT NULL CHECK (duration_minutes >= 0),
    energy_amount NUMERIC(12, 2) NOT NULL CHECK (energy_amount >= 0),
    service_amount NUMERIC(12, 2) NOT NULL CHECK (service_amount >= 0),
    occupancy_amount NUMERIC(12, 2) NOT NULL CHECK (occupancy_amount >= 0),
    total_amount NUMERIC(12, 2) NOT NULL CHECK (total_amount >= 0),
    currency TEXT NOT NULL DEFAULT 'CNY',
    status TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'confirmed', 'pending_review')),
    exception_flag BOOLEAN NOT NULL DEFAULT false,
    generated_by TEXT NOT NULL DEFAULT 'billing-engine',
    generated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ NULL,
    UNIQUE (session_id)
);

CREATE INDEX IF NOT EXISTS idx_billing_drafts_status ON billing_drafts(status, generated_at DESC);

CREATE TRIGGER trg_billing_drafts_updated_at
BEFORE UPDATE ON billing_drafts
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE IF NOT EXISTS reconciliation_exceptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    exception_no TEXT NOT NULL UNIQUE,
    bill_id UUID NOT NULL REFERENCES billing_drafts(id) ON DELETE CASCADE,
    session_id UUID NOT NULL REFERENCES charging_sessions(id) ON DELETE RESTRICT,
    exception_type TEXT NOT NULL CHECK (exception_type IN ('energy', 'duration', 'amount', 'stop_reason')),
    severity TEXT NOT NULL CHECK (severity IN ('low', 'medium', 'high', 'critical')),
    status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'reviewing', 'resolved')),
    reason TEXT NOT NULL,
    suggested_action TEXT NOT NULL DEFAULT '',
    detected_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    resolved_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ NULL
);

CREATE INDEX IF NOT EXISTS idx_reconciliation_exceptions_session_status ON reconciliation_exceptions(session_id, status);
CREATE INDEX IF NOT EXISTS idx_reconciliation_exceptions_status ON reconciliation_exceptions(status, detected_at DESC);

CREATE TRIGGER trg_reconciliation_exceptions_updated_at
BEFORE UPDATE ON reconciliation_exceptions
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE IF NOT EXISTS load_policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    site_id UUID NOT NULL REFERENCES sites(id) ON DELETE RESTRICT,
    area_id UUID NULL REFERENCES areas(id) ON DELETE RESTRICT,
    group_id UUID NULL REFERENCES charger_groups(id) ON DELETE RESTRICT,
    scope_type TEXT NOT NULL CHECK (scope_type IN ('site', 'area', 'group')),
    scope_code TEXT NOT NULL,
    scope_name TEXT NOT NULL,
    threshold_kw DOUBLE PRECISION NOT NULL CHECK (threshold_kw > 0),
    warning_kw DOUBLE PRECISION NOT NULL CHECK (warning_kw > 0),
    action_mode TEXT NOT NULL CHECK (action_mode IN ('limit_power', 'pause', 'queue', 'reject')),
    version INTEGER NOT NULL CHECK (version > 0),
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('draft', 'active', 'retired')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ NULL,
    UNIQUE (scope_type, scope_code, version)
);

CREATE INDEX IF NOT EXISTS idx_load_policies_site_status ON load_policies(site_id, status);

CREATE TRIGGER trg_load_policies_updated_at
BEFORE UPDATE ON load_policies
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE IF NOT EXISTS load_control_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    record_no TEXT NOT NULL UNIQUE,
    site_id UUID NOT NULL REFERENCES sites(id) ON DELETE RESTRICT,
    area_id UUID NULL REFERENCES areas(id) ON DELETE SET NULL,
    group_id UUID NULL REFERENCES charger_groups(id) ON DELETE SET NULL,
    session_id UUID NULL REFERENCES charging_sessions(id) ON DELETE SET NULL,
    connector_id UUID NULL REFERENCES connectors(id) ON DELETE SET NULL,
    action_type TEXT NOT NULL CHECK (
        action_type IN ('limit_power', 'pause', 'resume', 'queue', 'reject', 'promote', 'release')
    ),
    trigger_type TEXT NOT NULL DEFAULT 'manual' CHECK (trigger_type IN ('manual', 'auto')),
    reason TEXT NOT NULL,
    before_load_kw DOUBLE PRECISION NOT NULL DEFAULT 0 CHECK (before_load_kw >= 0),
    after_load_kw DOUBLE PRECISION NOT NULL DEFAULT 0 CHECK (after_load_kw >= 0),
    target_power_kw DOUBLE PRECISION NULL CHECK (target_power_kw IS NULL OR target_power_kw >= 0),
    status TEXT NOT NULL DEFAULT 'applied' CHECK (status IN ('recommended', 'sent', 'applied', 'rejected')),
    operator_name TEXT NOT NULL DEFAULT 'operations',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ NULL
);

CREATE INDEX IF NOT EXISTS idx_load_control_records_site_time ON load_control_records(site_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_load_control_records_session_time ON load_control_records(session_id, created_at DESC);

CREATE TRIGGER trg_load_control_records_updated_at
BEFORE UPDATE ON load_control_records
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

WITH active_site AS (
    SELECT id, code, name, load_limit_kw
    FROM sites
    WHERE code = 'HQ-CAMPUS' AND deleted_at IS NULL
),
upsert_policy AS (
    INSERT INTO pricing_policies (
        site_id, code, name, version, charger_type, effective_from, status
    )
    SELECT id, 'HQ-STD', '总部园区标准计价', 1, 'all', now() - interval '1 day', 'active'
    FROM active_site
    ON CONFLICT (site_id, code, version) DO UPDATE SET
        name = EXCLUDED.name,
        charger_type = EXCLUDED.charger_type,
        status = EXCLUDED.status
    RETURNING id
)
INSERT INTO pricing_periods (
    policy_id, label, start_minute, end_minute,
    energy_price_per_kwh, service_fee_per_kwh, occupancy_fee_per_minute
)
SELECT id, period.label, period.start_minute, period.end_minute,
       period.energy_price, period.service_fee, period.occupancy_fee
FROM upsert_policy
CROSS JOIN LATERAL (
    VALUES
        ('平段', 0, 420, 0.7800::numeric, 0.2200::numeric, 0.0200::numeric),
        ('峰段', 420, 1380, 1.1200::numeric, 0.2800::numeric, 0.0300::numeric),
        ('谷段', 1380, 1440, 0.5200::numeric, 0.1800::numeric, 0.0100::numeric)
) AS period(label, start_minute, end_minute, energy_price, service_fee, occupancy_fee)
WHERE NOT EXISTS (
    SELECT 1 FROM pricing_periods existing
    WHERE existing.policy_id = upsert_policy.id
      AND existing.start_minute = period.start_minute
      AND existing.end_minute = period.end_minute
);

INSERT INTO load_policies (
    site_id, scope_type, scope_code, scope_name, threshold_kw, warning_kw, action_mode, version, status
)
SELECT id, 'site', code, name, load_limit_kw, load_limit_kw * 0.9, 'limit_power', 1, 'active'
FROM (
    SELECT id, code, name, load_limit_kw
    FROM sites
    WHERE code = 'HQ-CAMPUS' AND deleted_at IS NULL
) AS active_site
ON CONFLICT (scope_type, scope_code, version) DO UPDATE SET
    threshold_kw = EXCLUDED.threshold_kw,
    warning_kw = EXCLUDED.warning_kw,
    action_mode = EXCLUDED.action_mode,
    status = EXCLUDED.status;

INSERT INTO load_policies (
    site_id, area_id, scope_type, scope_code, scope_name, threshold_kw, warning_kw, action_mode, version, status
)
SELECT site_id, id, 'area', code, name, load_limit_kw, load_limit_kw * 0.9, 'queue', 1, 'active'
FROM areas
WHERE deleted_at IS NULL
  AND site_id IN (
      SELECT id
      FROM sites
      WHERE code = 'HQ-CAMPUS' AND deleted_at IS NULL
  )
ON CONFLICT (scope_type, scope_code, version) DO UPDATE SET
    threshold_kw = EXCLUDED.threshold_kw,
    warning_kw = EXCLUDED.warning_kw,
    action_mode = EXCLUDED.action_mode,
    status = EXCLUDED.status;

INSERT INTO load_policies (
    site_id, area_id, group_id, scope_type, scope_code, scope_name, threshold_kw, warning_kw, action_mode, version, status
)
SELECT site_id, area_id, id, 'group', code, name, load_limit_kw, load_limit_kw * 0.9, 'pause', 1, 'active'
FROM charger_groups
WHERE deleted_at IS NULL
  AND site_id IN (
      SELECT id
      FROM sites
      WHERE code = 'HQ-CAMPUS' AND deleted_at IS NULL
  )
ON CONFLICT (scope_type, scope_code, version) DO UPDATE SET
    threshold_kw = EXCLUDED.threshold_kw,
    warning_kw = EXCLUDED.warning_kw,
    action_mode = EXCLUDED.action_mode,
    status = EXCLUDED.status;
