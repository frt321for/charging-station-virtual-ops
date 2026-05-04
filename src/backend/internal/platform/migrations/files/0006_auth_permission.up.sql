CREATE TABLE IF NOT EXISTS roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'disabled')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ NULL
);

CREATE TRIGGER trg_roles_updated_at
BEFORE UPDATE ON roles
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE IF NOT EXISTS permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS role_permissions (
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (role_id, permission_id)
);

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'disabled')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ NULL
);

CREATE INDEX IF NOT EXISTS idx_users_status ON users(status);

CREATE TRIGGER trg_users_updated_at
BEFORE UPDATE ON users
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE IF NOT EXISTS user_roles (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, role_id)
);

CREATE TABLE IF NOT EXISTS user_site_access (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    site_id UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    access_level TEXT NOT NULL DEFAULT 'member' CHECK (access_level IN ('member', 'manager')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, site_id)
);

CREATE INDEX IF NOT EXISTS idx_user_site_access_site ON user_site_access(site_id);

CREATE TABLE IF NOT EXISTS auth_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ NULL,
    user_agent TEXT NOT NULL DEFAULT '',
    ip_address TEXT NOT NULL DEFAULT '',
    last_seen_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_auth_sessions_user_time ON auth_sessions(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_auth_sessions_active ON auth_sessions(token_hash, expires_at)
WHERE revoked_at IS NULL;

ALTER TABLE audit_logs
    ADD COLUMN IF NOT EXISTS actor_user_id UUID NULL REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS actor_role_code TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS ip_address TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS trace_id TEXT NOT NULL DEFAULT '';

INSERT INTO roles (code, name, description)
VALUES
    ('operations', '运营人员', '配置站点运营策略、模拟器和平台配置'),
    ('station_manager', '站长', '查看站点状态并处理站点内运营动作'),
    ('maintenance', '运维人员', '处理故障工单、维修和复测'),
    ('finance', '财务人员', '核查账单、修正对账异常和导出报表'),
    ('support', '客服人员', '查询会话和异常信息，不允许控制命令'),
    ('ai_assistant', 'AI 运维助手', '只读解释、摘要和报告草稿')
ON CONFLICT (code) DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    status = 'active';

INSERT INTO permissions (code, name, description)
VALUES
    ('sites:any', '全部站点对象权限', '允许访问所有站点对象'),
    ('sites:read', '站点读取', '查看站点、区域、桩组和桩机拓扑'),
    ('sessions:read', '会话读取', '查看会话列表、详情和事件时间线'),
    ('sessions:write', '会话操作', '创建预约和推进会话状态'),
    ('commands:write', '远程控制', '下发启动、停止、暂停、恢复、复位和限功率命令'),
    ('simulator:control', '模拟器控制', '启动、停止或运行虚拟桩模拟器'),
    ('load_control:read', '负载控制读取', '查看负载策略、队列和控制记录'),
    ('load_control:write', '负载控制操作', '创建限功率、暂停、恢复、排队和释放记录'),
    ('billing:read', '账单读取', '查看价格策略、账单草稿和对账异常'),
    ('billing:generate', '账单生成', '根据会话生成账单草稿'),
    ('finance:review', '财务核查', '核查异常、创建修正、确认账单和导出对账'),
    ('maintenance:read', '维保读取', '查看故障、工单和 SLA'),
    ('maintenance:write', '维保操作', '创建工单和推进维修状态'),
    ('audit:read', '审计读取', '查看远程控制、财务和权限敏感审计'),
    ('config:manage', '配置管理', '维护站点、策略和平台配置'),
    ('users:manage', '用户角色管理', '维护用户、角色和授权边界'),
    ('ai:read', 'AI 只读助手', '访问 AI 解释、摘要和报告草稿')
ON CONFLICT (code) DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description;

WITH role_permission_seed(role_code, permission_code) AS (
    VALUES
        ('operations', 'sites:any'),
        ('operations', 'sites:read'),
        ('operations', 'sessions:read'),
        ('operations', 'sessions:write'),
        ('operations', 'commands:write'),
        ('operations', 'simulator:control'),
        ('operations', 'load_control:read'),
        ('operations', 'load_control:write'),
        ('operations', 'billing:read'),
        ('operations', 'billing:generate'),
        ('operations', 'finance:review'),
        ('operations', 'maintenance:read'),
        ('operations', 'maintenance:write'),
        ('operations', 'audit:read'),
        ('operations', 'config:manage'),
        ('operations', 'users:manage'),
        ('operations', 'ai:read'),
        ('station_manager', 'sites:read'),
        ('station_manager', 'sessions:read'),
        ('station_manager', 'sessions:write'),
        ('station_manager', 'commands:write'),
        ('station_manager', 'load_control:read'),
        ('station_manager', 'load_control:write'),
        ('station_manager', 'billing:read'),
        ('station_manager', 'maintenance:read'),
        ('station_manager', 'audit:read'),
        ('maintenance', 'sites:read'),
        ('maintenance', 'sessions:read'),
        ('maintenance', 'maintenance:read'),
        ('maintenance', 'maintenance:write'),
        ('finance', 'sites:read'),
        ('finance', 'sessions:read'),
        ('finance', 'billing:read'),
        ('finance', 'billing:generate'),
        ('finance', 'finance:review'),
        ('finance', 'audit:read'),
        ('support', 'sites:read'),
        ('support', 'sessions:read'),
        ('support', 'billing:read'),
        ('support', 'maintenance:read'),
        ('ai_assistant', 'sites:read'),
        ('ai_assistant', 'sessions:read'),
        ('ai_assistant', 'billing:read'),
        ('ai_assistant', 'maintenance:read'),
        ('ai_assistant', 'audit:read'),
        ('ai_assistant', 'ai:read')
)
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM role_permission_seed seed
INNER JOIN roles r ON r.code = seed.role_code
INNER JOIN permissions p ON p.code = seed.permission_code
ON CONFLICT DO NOTHING;

WITH user_seed(username, display_name, role_code) AS (
    VALUES
        ('ops.admin', '运营管理员', 'operations'),
        ('station.manager', '总部园区站长', 'station_manager'),
        ('maintenance.tech', '维保一班', 'maintenance'),
        ('finance.reviewer', '财务核查员', 'finance'),
        ('support.agent', '客服坐席', 'support'),
        ('ai.assistant', 'AI 运维助手', 'ai_assistant')
),
upsert_users AS (
    INSERT INTO users (username, display_name, password_hash, status)
    SELECT username, display_name, crypt('Password2026!', gen_salt('bf', 10)), 'active'
    FROM user_seed
    ON CONFLICT (username) DO UPDATE SET
        display_name = EXCLUDED.display_name,
        status = 'active'
    RETURNING id, username
)
INSERT INTO user_roles (user_id, role_id)
SELECT u.id, r.id
FROM upsert_users u
INNER JOIN user_seed seed ON seed.username = u.username
INNER JOIN roles r ON r.code = seed.role_code
ON CONFLICT DO NOTHING;

WITH scoped_users AS (
    SELECT u.id AS user_id
    FROM users u
    WHERE u.username IN (
        'station.manager',
        'maintenance.tech',
        'finance.reviewer',
        'support.agent',
        'ai.assistant'
    )
),
default_site AS (
    SELECT id AS site_id
    FROM sites
    WHERE code = 'HQ-CAMPUS' AND deleted_at IS NULL
)
INSERT INTO user_site_access (user_id, site_id, access_level)
SELECT scoped_users.user_id, default_site.site_id, 'member'
FROM scoped_users
CROSS JOIN default_site
ON CONFLICT DO NOTHING;
