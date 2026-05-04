DELETE FROM charger_meter_values
WHERE session_id IN (
    SELECT id FROM charging_sessions WHERE session_no = 'CS-20260504-0001'
);

DELETE FROM session_events
WHERE session_id IN (
    SELECT id FROM charging_sessions WHERE session_no = 'CS-20260504-0001'
);

DELETE FROM charging_sessions
WHERE session_no = 'CS-20260504-0001';

DELETE FROM connectors
WHERE code IN ('AC-N-001-01', 'AC-N-001-02', 'DC-S-001-01', 'DC-S-001-02');

DELETE FROM chargers
WHERE code IN ('AC-N-001', 'DC-S-001');

DELETE FROM charger_groups
WHERE code IN ('N-A1', 'S-B1');

DELETE FROM areas
WHERE code IN ('NORTH', 'SOUTH');

DELETE FROM sites
WHERE code = 'HQ-CAMPUS';
