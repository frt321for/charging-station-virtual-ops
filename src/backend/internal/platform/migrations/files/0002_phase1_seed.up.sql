WITH inserted_site AS (
    INSERT INTO sites (
        code, name, campus, capacity_kw, load_limit_kw, timezone,
        sla_response_minutes, sla_recovery_minutes, status
    )
    VALUES (
        'HQ-CAMPUS', '总部园区充电站', '总部园区', 720, 620, 'Asia/Shanghai',
        15, 120, 'active'
    )
    ON CONFLICT (code) DO UPDATE SET
        name = EXCLUDED.name,
        campus = EXCLUDED.campus,
        capacity_kw = EXCLUDED.capacity_kw,
        load_limit_kw = EXCLUDED.load_limit_kw
    RETURNING id
),
north_area AS (
    INSERT INTO areas (site_id, code, name, load_limit_kw, sort_order, status)
    SELECT id, 'NORTH', '北区车棚', 280, 10, 'active' FROM inserted_site
    ON CONFLICT (site_id, code) DO UPDATE SET
        name = EXCLUDED.name,
        load_limit_kw = EXCLUDED.load_limit_kw
    RETURNING id, site_id
),
south_area AS (
    INSERT INTO areas (site_id, code, name, load_limit_kw, sort_order, status)
    SELECT id, 'SOUTH', '南区地下库', 340, 20, 'active' FROM inserted_site
    ON CONFLICT (site_id, code) DO UPDATE SET
        name = EXCLUDED.name,
        load_limit_kw = EXCLUDED.load_limit_kw
    RETURNING id, site_id
),
north_group AS (
    INSERT INTO charger_groups (site_id, area_id, code, name, electrical_node, load_limit_kw, priority, status)
    SELECT site_id, id, 'N-A1', '北区 A 回路', 'TX-01/N-A1', 180, 10, 'active' FROM north_area
    ON CONFLICT (site_id, code) DO UPDATE SET
        name = EXCLUDED.name,
        electrical_node = EXCLUDED.electrical_node,
        load_limit_kw = EXCLUDED.load_limit_kw
    RETURNING id, site_id
),
south_group AS (
    INSERT INTO charger_groups (site_id, area_id, code, name, electrical_node, load_limit_kw, priority, status)
    SELECT site_id, id, 'S-B1', '南区 B 回路', 'TX-02/S-B1', 240, 20, 'active' FROM south_area
    ON CONFLICT (site_id, code) DO UPDATE SET
        name = EXCLUDED.name,
        electrical_node = EXCLUDED.electrical_node,
        load_limit_kw = EXCLUDED.load_limit_kw
    RETURNING id, site_id
),
ac_charger AS (
    INSERT INTO chargers (
        group_id, code, name, charger_type, rated_power_kw, connector_count,
        status, installation_location, last_heartbeat_at
    )
    SELECT id, 'AC-N-001', '北区 1 号交流桩', 'ac', 14, 2, 'charging', '北区车棚 N-01', now() - interval '25 seconds'
    FROM north_group
    ON CONFLICT (code) DO UPDATE SET
        status = EXCLUDED.status,
        last_heartbeat_at = EXCLUDED.last_heartbeat_at
    RETURNING id, group_id
),
dc_charger AS (
    INSERT INTO chargers (
        group_id, code, name, charger_type, rated_power_kw, connector_count,
        status, installation_location, last_heartbeat_at
    )
    SELECT id, 'DC-S-001', '南区 1 号直流桩', 'dc', 120, 2, 'available', '南区地下库 B1-03', now() - interval '18 seconds'
    FROM south_group
    ON CONFLICT (code) DO UPDATE SET
        status = EXCLUDED.status,
        last_heartbeat_at = EXCLUDED.last_heartbeat_at
    RETURNING id, group_id
),
ac_connector_1 AS (
    INSERT INTO connectors (charger_id, connector_no, code, max_power_kw, status)
    SELECT id, 1, 'AC-N-001-01', 7, 'charging' FROM ac_charger
    ON CONFLICT (code) DO UPDATE SET status = EXCLUDED.status
    RETURNING id, charger_id
),
ac_connector_2 AS (
    INSERT INTO connectors (charger_id, connector_no, code, max_power_kw, status)
    SELECT id, 2, 'AC-N-001-02', 7, 'available' FROM ac_charger
    ON CONFLICT (code) DO UPDATE SET status = EXCLUDED.status
    RETURNING id
),
dc_connector_1 AS (
    INSERT INTO connectors (charger_id, connector_no, code, max_power_kw, status)
    SELECT id, 1, 'DC-S-001-01', 60, 'available' FROM dc_charger
    ON CONFLICT (code) DO UPDATE SET status = EXCLUDED.status
    RETURNING id
),
dc_connector_2 AS (
    INSERT INTO connectors (charger_id, connector_no, code, max_power_kw, status)
    SELECT id, 2, 'DC-S-001-02', 60, 'available' FROM dc_charger
    ON CONFLICT (code) DO UPDATE SET status = EXCLUDED.status
    RETURNING id
),
active_session AS (
    INSERT INTO charging_sessions (
        session_no, site_id, charger_id, connector_id, status,
        reservation_expires_at, started_at, meter_start_kwh
    )
    SELECT
        'CS-20260504-0001',
        s.id,
        c.charger_id,
        c.id,
        'charging',
        now() + interval '30 minutes',
        now() - interval '18 minutes',
        1240.62
    FROM inserted_site s
    CROSS JOIN ac_connector_1 c
    ON CONFLICT (session_no) DO UPDATE SET
        status = EXCLUDED.status,
        started_at = EXCLUDED.started_at,
        meter_start_kwh = EXCLUDED.meter_start_kwh
    RETURNING id, charger_id, connector_id
)
INSERT INTO session_events (session_id, event_type, occurred_at, source, payload)
SELECT id, event_type, occurred_at, source, payload
FROM active_session
CROSS JOIN LATERAL (
    VALUES
        ('ReservationCreated', now() - interval '45 minutes', 'operations', '{"reservationRule":"standard-30m"}'::jsonb),
        ('PlugInDetected', now() - interval '22 minutes', 'protocol-gateway', '{"connectorCode":"AC-N-001-01"}'::jsonb),
        ('RemoteStartSent', now() - interval '20 minutes', 'operations', '{"command":"start","attempt":1}'::jsonb),
        ('StartAccepted', now() - interval '19 minutes', 'protocol-gateway', '{"receipt":"accepted"}'::jsonb),
        ('MeterValueReceived', now() - interval '3 minutes', 'protocol-gateway', '{"meterKwh":1242.76,"powerKw":6.8}'::jsonb)
) AS events(event_type, occurred_at, source, payload)
WHERE NOT EXISTS (
    SELECT 1
    FROM session_events existing
    WHERE existing.session_id = active_session.id
      AND existing.event_type = events.event_type
);

INSERT INTO charger_meter_values (
    time, charger_id, connector_id, session_id, power_kw, voltage_v, current_a, meter_kwh, raw_payload
)
SELECT now() - interval '3 minutes', charger_id, connector_id, id, 6.8, 226, 30.1, 1242.76, '{"source":"seed"}'::jsonb
FROM charging_sessions
WHERE session_no = 'CS-20260504-0001'
ON CONFLICT DO NOTHING;
