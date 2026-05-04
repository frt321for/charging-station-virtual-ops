ALTER TABLE audit_logs
    DROP COLUMN IF EXISTS trace_id,
    DROP COLUMN IF EXISTS ip_address,
    DROP COLUMN IF EXISTS actor_role_code,
    DROP COLUMN IF EXISTS actor_user_id;

DROP TABLE IF EXISTS auth_sessions;
DROP TABLE IF EXISTS user_site_access;
DROP TABLE IF EXISTS user_roles;
DROP TABLE IF EXISTS role_permissions;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS permissions;
DROP TABLE IF EXISTS roles;
