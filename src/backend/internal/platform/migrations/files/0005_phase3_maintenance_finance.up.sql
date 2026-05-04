CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_type TEXT NOT NULL,
    entity_id UUID NULL,
    action TEXT NOT NULL,
    actor_name TEXT NOT NULL DEFAULT 'system',
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_entity ON audit_logs(entity_type, entity_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action_time ON audit_logs(action, created_at DESC);

CREATE TABLE IF NOT EXISTS work_orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    work_order_no TEXT NOT NULL UNIQUE,
    fault_id UUID NULL REFERENCES charger_faults(id) ON DELETE SET NULL,
    site_id UUID NOT NULL REFERENCES sites(id) ON DELETE RESTRICT,
    charger_id UUID NOT NULL REFERENCES chargers(id) ON DELETE RESTRICT,
    connector_id UUID NULL REFERENCES connectors(id) ON DELETE SET NULL,
    session_id UUID NULL REFERENCES charging_sessions(id) ON DELETE SET NULL,
    severity TEXT NOT NULL CHECK (severity IN ('low', 'medium', 'high', 'critical')),
    status TEXT NOT NULL DEFAULT 'open' CHECK (
        status IN ('open', 'assigned', 'accepted', 'arrived', 'handling', 'retest', 'recovered', 'closed', 'cancelled')
    ),
    impact_scope TEXT NOT NULL DEFAULT 'charger',
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    assignee_name TEXT NOT NULL DEFAULT '',
    response_due_at TIMESTAMPTZ NOT NULL,
    recovery_due_at TIMESTAMPTZ NOT NULL,
    accepted_at TIMESTAMPTZ NULL,
    arrived_at TIMESTAMPTZ NULL,
    handling_at TIMESTAMPTZ NULL,
    retest_at TIMESTAMPTZ NULL,
    recovered_at TIMESTAMPTZ NULL,
    closed_at TIMESTAMPTZ NULL,
    sla_response_breached BOOLEAN NOT NULL DEFAULT false,
    sla_recovery_breached BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ NULL
);

CREATE INDEX IF NOT EXISTS idx_work_orders_site_status ON work_orders(site_id, status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_work_orders_fault ON work_orders(fault_id);
CREATE INDEX IF NOT EXISTS idx_work_orders_sla ON work_orders(status, response_due_at, recovery_due_at);

CREATE TRIGGER trg_work_orders_updated_at
BEFORE UPDATE ON work_orders
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE IF NOT EXISTS work_order_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    work_order_id UUID NOT NULL REFERENCES work_orders(id) ON DELETE CASCADE,
    event_type TEXT NOT NULL,
    from_status TEXT NOT NULL DEFAULT '',
    to_status TEXT NOT NULL DEFAULT '',
    actor_name TEXT NOT NULL DEFAULT 'operations',
    note TEXT NOT NULL DEFAULT '',
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_work_order_events_order_time ON work_order_events(work_order_id, occurred_at, id);

CREATE TABLE IF NOT EXISTS billing_corrections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    correction_no TEXT NOT NULL UNIQUE,
    bill_id UUID NOT NULL REFERENCES billing_drafts(id) ON DELETE CASCADE,
    exception_id UUID NULL REFERENCES reconciliation_exceptions(id) ON DELETE SET NULL,
    corrected_energy_kwh NUMERIC(12, 3) NULL CHECK (corrected_energy_kwh IS NULL OR corrected_energy_kwh >= 0),
    corrected_duration_minutes INTEGER NULL CHECK (corrected_duration_minutes IS NULL OR corrected_duration_minutes >= 0),
    corrected_total_amount NUMERIC(12, 2) NULL CHECK (corrected_total_amount IS NULL OR corrected_total_amount >= 0),
    reason TEXT NOT NULL,
    reviewer_name TEXT NOT NULL DEFAULT 'finance-reviewer',
    status TEXT NOT NULL DEFAULT 'applied' CHECK (status IN ('draft', 'applied', 'voided')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_billing_corrections_bill_time ON billing_corrections(bill_id, created_at DESC);

WITH target AS (
    SELECT
        s.id AS site_id,
        c.id AS charger_id,
        cn.id AS connector_id
    FROM sites s
    INNER JOIN charger_groups g ON g.site_id = s.id
    INNER JOIN chargers c ON c.group_id = g.id
    INNER JOIN connectors cn ON cn.charger_id = c.id
    WHERE s.code = 'HQ-CAMPUS'
      AND cn.code = 'DC-S-001-02'
      AND s.deleted_at IS NULL
      AND c.deleted_at IS NULL
      AND cn.deleted_at IS NULL
    LIMIT 1
),
seed_fault AS (
    INSERT INTO charger_faults (
        fault_no, charger_id, connector_id, fault_code, severity, status, occurred_at, payload
    )
    SELECT
        'FT-SEED-HQ-001',
        charger_id,
        connector_id,
        'CONNECTOR_UNAVAILABLE',
        'high',
        'open',
        now() - interval '18 minutes',
        '{"source":"phase3-seed"}'::jsonb
    FROM target
    ON CONFLICT (fault_no) DO NOTHING
    RETURNING id, charger_id, connector_id
),
fault_row AS (
    SELECT id, charger_id, connector_id
    FROM seed_fault
    UNION ALL
    SELECT id, charger_id, connector_id
    FROM charger_faults
    WHERE fault_no = 'FT-SEED-HQ-001'
    LIMIT 1
),
seed_order AS (
    INSERT INTO work_orders (
        work_order_no, fault_id, site_id, charger_id, connector_id,
        severity, status, impact_scope, title, description, assignee_name,
        response_due_at, recovery_due_at
    )
    SELECT
        'WO-SEED-HQ-001',
        f.id,
        t.site_id,
        f.charger_id,
        f.connector_id,
        'high',
        'assigned',
        'connector',
        '直流桩 2 号枪不可用',
        'CONNECTOR_UNAVAILABLE',
        'maintenance-shift-a',
        now() - interval '3 minutes',
        now() + interval '102 minutes'
    FROM fault_row f
    CROSS JOIN target t
    ON CONFLICT (work_order_no) DO NOTHING
    RETURNING id, fault_id
)
UPDATE charger_faults cf
SET status = 'linked_work_order'
FROM seed_order so
WHERE cf.id = so.fault_id;

UPDATE chargers
SET status = 'fault'
WHERE id IN (
    SELECT charger_id
    FROM charger_faults
    WHERE fault_no = 'FT-SEED-HQ-001'
);

UPDATE connectors
SET status = 'fault'
WHERE id IN (
    SELECT connector_id
    FROM charger_faults
    WHERE fault_no = 'FT-SEED-HQ-001'
);
