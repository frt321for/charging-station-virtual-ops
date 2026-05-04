CREATE TABLE IF NOT EXISTS charger_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    charger_id UUID NOT NULL REFERENCES chargers(id) ON DELETE RESTRICT,
    connector_id UUID NULL REFERENCES connectors(id) ON DELETE SET NULL,
    session_id UUID NULL REFERENCES charging_sessions(id) ON DELETE SET NULL,
    event_type TEXT NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    source TEXT NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_charger_events_charger_time ON charger_events(charger_id, occurred_at DESC);
CREATE INDEX IF NOT EXISTS idx_charger_events_session_time ON charger_events(session_id, occurred_at DESC);
CREATE INDEX IF NOT EXISTS idx_charger_events_type_time ON charger_events(event_type, occurred_at DESC);

CREATE TABLE IF NOT EXISTS remote_commands (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    command_no TEXT NOT NULL UNIQUE,
    session_id UUID NULL REFERENCES charging_sessions(id) ON DELETE SET NULL,
    charger_id UUID NOT NULL REFERENCES chargers(id) ON DELETE RESTRICT,
    connector_id UUID NULL REFERENCES connectors(id) ON DELETE SET NULL,
    command_type TEXT NOT NULL CHECK (
        command_type IN ('start', 'stop', 'pause', 'resume', 'reset', 'limit_power')
    ),
    status TEXT NOT NULL DEFAULT 'sent' CHECK (
        status IN ('sent', 'accepted', 'rejected', 'timeout', 'failed', 'retried')
    ),
    requested_by TEXT NOT NULL DEFAULT 'operations',
    target_power_kw DOUBLE PRECISION NULL CHECK (target_power_kw IS NULL OR target_power_kw >= 0),
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    result_message TEXT NOT NULL DEFAULT '',
    sent_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    acknowledged_at TIMESTAMPTZ NULL,
    retry_of UUID NULL REFERENCES remote_commands(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ NULL
);

CREATE INDEX IF NOT EXISTS idx_remote_commands_session_time ON remote_commands(session_id, sent_at DESC);
CREATE INDEX IF NOT EXISTS idx_remote_commands_charger_time ON remote_commands(charger_id, sent_at DESC);
CREATE INDEX IF NOT EXISTS idx_remote_commands_status ON remote_commands(status);

CREATE TRIGGER trg_remote_commands_updated_at
BEFORE UPDATE ON remote_commands
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE IF NOT EXISTS charger_faults (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    fault_no TEXT NOT NULL UNIQUE,
    charger_id UUID NOT NULL REFERENCES chargers(id) ON DELETE RESTRICT,
    connector_id UUID NULL REFERENCES connectors(id) ON DELETE SET NULL,
    session_id UUID NULL REFERENCES charging_sessions(id) ON DELETE SET NULL,
    fault_code TEXT NOT NULL,
    severity TEXT NOT NULL CHECK (severity IN ('low', 'medium', 'high', 'critical')),
    status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'linked_work_order', 'resolved')),
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    resolved_at TIMESTAMPTZ NULL,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ NULL
);

CREATE INDEX IF NOT EXISTS idx_charger_faults_charger_status ON charger_faults(charger_id, status);
CREATE INDEX IF NOT EXISTS idx_charger_faults_session_time ON charger_faults(session_id, occurred_at DESC);

CREATE TRIGGER trg_charger_faults_updated_at
BEFORE UPDATE ON charger_faults
FOR EACH ROW EXECUTE FUNCTION set_updated_at();
