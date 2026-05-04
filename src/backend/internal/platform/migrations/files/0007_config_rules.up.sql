CREATE TABLE IF NOT EXISTS reservation_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    site_id UUID NOT NULL REFERENCES sites(id) ON DELETE RESTRICT,
    code TEXT NOT NULL,
    name TEXT NOT NULL,
    hold_minutes INTEGER NOT NULL CHECK (hold_minutes > 0),
    timeout_action TEXT NOT NULL CHECK (timeout_action IN ('release', 'promote_queue', 'cancel')),
    version INTEGER NOT NULL CHECK (version > 0),
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('draft', 'active', 'retired')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ NULL,
    UNIQUE (site_id, code, version)
);

CREATE TRIGGER trg_reservation_rules_updated_at
BEFORE UPDATE ON reservation_rules
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE IF NOT EXISTS queue_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    site_id UUID NOT NULL REFERENCES sites(id) ON DELETE RESTRICT,
    code TEXT NOT NULL,
    name TEXT NOT NULL,
    strategy TEXT NOT NULL CHECK (strategy IN ('reservation_time', 'member_priority', 'load_priority')),
    max_queue_size INTEGER NOT NULL CHECK (max_queue_size > 0),
    priority_factor TEXT NOT NULL DEFAULT 'reservation_time',
    version INTEGER NOT NULL CHECK (version > 0),
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('draft', 'active', 'retired')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ NULL,
    UNIQUE (site_id, code, version)
);

CREATE TRIGGER trg_queue_rules_updated_at
BEFORE UPDATE ON queue_rules
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

WITH active_site AS (
    SELECT id
    FROM sites
    WHERE code = 'HQ-CAMPUS' AND deleted_at IS NULL
)
INSERT INTO reservation_rules (
    site_id, code, name, hold_minutes, timeout_action, version, status
)
SELECT id, 'HQ-RESERVE-STD', '总部预约保留规则', 30, 'promote_queue', 1, 'active'
FROM active_site
ON CONFLICT (site_id, code, version) DO UPDATE SET
    name = EXCLUDED.name,
    hold_minutes = EXCLUDED.hold_minutes,
    timeout_action = EXCLUDED.timeout_action,
    status = EXCLUDED.status;

WITH active_site AS (
    SELECT id
    FROM sites
    WHERE code = 'HQ-CAMPUS' AND deleted_at IS NULL
)
INSERT INTO queue_rules (
    site_id, code, name, strategy, max_queue_size, priority_factor, version, status
)
SELECT id, 'HQ-QUEUE-PEAK', '总部高峰排队规则', 'reservation_time', 80, 'reservation_time+load_margin', 1, 'active'
FROM active_site
ON CONFLICT (site_id, code, version) DO UPDATE SET
    name = EXCLUDED.name,
    strategy = EXCLUDED.strategy,
    max_queue_size = EXCLUDED.max_queue_size,
    priority_factor = EXCLUDED.priority_factor,
    status = EXCLUDED.status;
