CREATE EXTENSION IF NOT EXISTS timescaledb;
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TABLE IF NOT EXISTS sites (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    campus TEXT NOT NULL,
    capacity_kw DOUBLE PRECISION NOT NULL CHECK (capacity_kw > 0),
    load_limit_kw DOUBLE PRECISION NOT NULL CHECK (load_limit_kw > 0),
    timezone TEXT NOT NULL DEFAULT 'Asia/Shanghai',
    sla_response_minutes INTEGER NOT NULL DEFAULT 15 CHECK (sla_response_minutes > 0),
    sla_recovery_minutes INTEGER NOT NULL DEFAULT 120 CHECK (sla_recovery_minutes > 0),
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ NULL
);

CREATE TRIGGER trg_sites_updated_at
BEFORE UPDATE ON sites
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE IF NOT EXISTS areas (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    site_id UUID NOT NULL REFERENCES sites(id) ON DELETE RESTRICT,
    code TEXT NOT NULL,
    name TEXT NOT NULL,
    load_limit_kw DOUBLE PRECISION NOT NULL CHECK (load_limit_kw > 0),
    sort_order INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ NULL,
    UNIQUE (site_id, code)
);

CREATE TRIGGER trg_areas_updated_at
BEFORE UPDATE ON areas
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE IF NOT EXISTS charger_groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    site_id UUID NOT NULL REFERENCES sites(id) ON DELETE RESTRICT,
    area_id UUID NOT NULL REFERENCES areas(id) ON DELETE RESTRICT,
    code TEXT NOT NULL,
    name TEXT NOT NULL,
    electrical_node TEXT NOT NULL,
    load_limit_kw DOUBLE PRECISION NOT NULL CHECK (load_limit_kw > 0),
    priority INTEGER NOT NULL DEFAULT 100,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ NULL,
    UNIQUE (site_id, code)
);

CREATE TRIGGER trg_charger_groups_updated_at
BEFORE UPDATE ON charger_groups
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE IF NOT EXISTS chargers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id UUID NOT NULL REFERENCES charger_groups(id) ON DELETE RESTRICT,
    code TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    charger_type TEXT NOT NULL CHECK (charger_type IN ('ac', 'dc')),
    rated_power_kw DOUBLE PRECISION NOT NULL CHECK (rated_power_kw > 0),
    connector_count INTEGER NOT NULL CHECK (connector_count > 0),
    status TEXT NOT NULL DEFAULT 'unregistered' CHECK (
        status IN ('unregistered', 'available', 'occupied', 'charging', 'fault', 'offline', 'maintenance', 'disabled')
    ),
    installation_location TEXT NOT NULL,
    maintenance_tag TEXT NOT NULL DEFAULT '',
    last_heartbeat_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ NULL
);

CREATE INDEX IF NOT EXISTS idx_chargers_group_status ON chargers(group_id, status);

CREATE TRIGGER trg_chargers_updated_at
BEFORE UPDATE ON chargers
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE IF NOT EXISTS connectors (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    charger_id UUID NOT NULL REFERENCES chargers(id) ON DELETE RESTRICT,
    connector_no INTEGER NOT NULL CHECK (connector_no > 0),
    code TEXT NOT NULL UNIQUE,
    max_power_kw DOUBLE PRECISION NOT NULL CHECK (max_power_kw > 0),
    status TEXT NOT NULL DEFAULT 'available' CHECK (
        status IN ('available', 'reserved', 'plugged', 'charging', 'fault', 'offline', 'maintenance', 'disabled')
    ),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ NULL,
    UNIQUE (charger_id, connector_no)
);

CREATE INDEX IF NOT EXISTS idx_connectors_charger_status ON connectors(charger_id, status);

CREATE TRIGGER trg_connectors_updated_at
BEFORE UPDATE ON connectors
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE IF NOT EXISTS charging_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_no TEXT NOT NULL UNIQUE,
    site_id UUID NOT NULL REFERENCES sites(id) ON DELETE RESTRICT,
    charger_id UUID NOT NULL REFERENCES chargers(id) ON DELETE RESTRICT,
    connector_id UUID NOT NULL REFERENCES connectors(id) ON DELETE RESTRICT,
    status TEXT NOT NULL CHECK (
        status IN (
            'reserved', 'waiting_arrival', 'plugged_in', 'starting', 'charging',
            'paused', 'stopping', 'pending_billing', 'billed', 'pending_review', 'cancelled'
        )
    ),
    reservation_expires_at TIMESTAMPTZ NULL,
    started_at TIMESTAMPTZ NULL,
    stopped_at TIMESTAMPTZ NULL,
    stop_reason TEXT NOT NULL DEFAULT '',
    meter_start_kwh DOUBLE PRECISION NULL CHECK (meter_start_kwh IS NULL OR meter_start_kwh >= 0),
    meter_stop_kwh DOUBLE PRECISION NULL CHECK (meter_stop_kwh IS NULL OR meter_stop_kwh >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ NULL
);

CREATE INDEX IF NOT EXISTS idx_charging_sessions_status ON charging_sessions(status);
CREATE INDEX IF NOT EXISTS idx_charging_sessions_site_updated ON charging_sessions(site_id, updated_at DESC);

CREATE TRIGGER trg_charging_sessions_updated_at
BEFORE UPDATE ON charging_sessions
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE IF NOT EXISTS session_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES charging_sessions(id) ON DELETE CASCADE,
    event_type TEXT NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    source TEXT NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_session_events_session_time ON session_events(session_id, occurred_at, id);
CREATE INDEX IF NOT EXISTS idx_session_events_type_time ON session_events(event_type, occurred_at DESC);

CREATE TABLE IF NOT EXISTS charger_meter_values (
    time TIMESTAMPTZ NOT NULL,
    charger_id UUID NOT NULL REFERENCES chargers(id) ON DELETE RESTRICT,
    connector_id UUID NOT NULL REFERENCES connectors(id) ON DELETE RESTRICT,
    session_id UUID NULL REFERENCES charging_sessions(id) ON DELETE SET NULL,
    power_kw DOUBLE PRECISION NOT NULL CHECK (power_kw >= 0),
    voltage_v DOUBLE PRECISION NULL CHECK (voltage_v IS NULL OR voltage_v >= 0),
    current_a DOUBLE PRECISION NULL CHECK (current_a IS NULL OR current_a >= 0),
    meter_kwh DOUBLE PRECISION NOT NULL CHECK (meter_kwh >= 0),
    raw_payload JSONB NOT NULL DEFAULT '{}'::jsonb
);

SELECT create_hypertable('charger_meter_values', 'time', if_not_exists => TRUE);

CREATE INDEX IF NOT EXISTS idx_charger_meter_values_session_time ON charger_meter_values(session_id, time DESC);
CREATE INDEX IF NOT EXISTS idx_charger_meter_values_charger_time ON charger_meter_values(charger_id, time DESC);
