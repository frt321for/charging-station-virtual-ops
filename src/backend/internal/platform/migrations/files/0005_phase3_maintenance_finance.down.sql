UPDATE connectors
SET status = 'available'
WHERE id IN (
    SELECT connector_id
    FROM charger_faults
    WHERE fault_no = 'FT-SEED-HQ-001'
);

UPDATE chargers
SET status = 'available'
WHERE id IN (
    SELECT charger_id
    FROM charger_faults
    WHERE fault_no = 'FT-SEED-HQ-001'
);

DELETE FROM charger_faults WHERE fault_no = 'FT-SEED-HQ-001';

DROP TABLE IF EXISTS billing_corrections;
DROP TABLE IF EXISTS work_order_events;
DROP TABLE IF EXISTS work_orders;
DROP TABLE IF EXISTS audit_logs;
