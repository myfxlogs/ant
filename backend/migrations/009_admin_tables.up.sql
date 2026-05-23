-- 009_admin_tables.up.sql
-- 管理员相关表

-- 权限表
CREATE TABLE IF NOT EXISTS permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(50) UNIQUE NOT NULL,
    name VARCHAR(100) NOT NULL,
    description VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 角色权限关联表
CREATE TABLE IF NOT EXISTS role_permissions (
    role VARCHAR(20) NOT NULL,
    permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    granted_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (role, permission_id)
);

-- 操作日志表
CREATE TABLE IF NOT EXISTS admin_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    
    module VARCHAR(50) NOT NULL,
    action_type VARCHAR(100) NOT NULL,
    target_type VARCHAR(50),
    target_id VARCHAR(100),
    
    ip_address VARCHAR(45),
    user_agent TEXT,
    request_method VARCHAR(10),
    request_path VARCHAR(255),
    
    details JSONB,
    
    success BOOLEAN DEFAULT TRUE,
    error_message VARCHAR(500),
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_admin_logs_user ON admin_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_admin_logs_module ON admin_logs(module);
CREATE INDEX IF NOT EXISTS idx_admin_logs_created ON admin_logs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_admin_logs_action_type ON admin_logs(action_type);

-- 预置权限数据
INSERT INTO permissions (code, name, description) VALUES
-- 用户管理
('user:read', '查看用户', '查看用户列表和详情'),
('user:create', '创建用户', '创建新用户'),
('user:update', '更新用户', '更新用户信息'),
('user:delete', '删除用户', '删除用户'),
-- MT账户管理
('account:read', '查看账户', '查看MT账户列表和详情'),
('account:create', '创建账户', '绑定新MT账户'),
('account:update', '更新账户', '更新账户信息'),
('account:delete', '删除账户', '删除MT账户'),
('account:connect', '连接账户', '连接MT服务器'),
('account:disconnect', '断开账户', '断开MT连接'),
-- 交易权限
('trade:execute', '执行交易', '下单、改单、平仓'),
('trade:view', '查看交易', '查看持仓和订单'),
('trade:analysis', '交易分析', '查看交易统计和分析'),
-- 行情权限
('market:read', '查看行情', '查看实时行情和K线'),
('market:subscribe', '订阅行情', '订阅品种'),
-- 管理员权限
('admin:view', '管理后台', '访问管理后台'),
('admin:logs', '日志查看', '查看操作日志'),
('admin:config', '系统配置', '修改系统配置')
ON CONFLICT (code) DO NOTHING;

-- 预置角色权限
INSERT INTO role_permissions (role, permission_id)
SELECT 'super_admin', id FROM permissions
ON CONFLICT (role, permission_id) DO NOTHING;

INSERT INTO role_permissions (role, permission_id)
SELECT 'operation', id FROM permissions WHERE code IN (
    'user:read', 'user:create', 'user:update',
    'account:read', 'account:create', 'account:update', 'account:delete',
    'account:connect', 'account:disconnect',
    'trade:execute', 'trade:view', 'trade:analysis',
    'market:read', 'market:subscribe',
    'admin:view', 'admin:logs'
)
ON CONFLICT (role, permission_id) DO NOTHING;

INSERT INTO role_permissions (role, permission_id)
SELECT 'customer_service', id FROM permissions WHERE code IN (
    'user:read', 'account:read',
    'trade:view', 'market:read'
)
ON CONFLICT (role, permission_id) DO NOTHING;

INSERT INTO role_permissions (role, permission_id)
SELECT 'audit', id FROM permissions WHERE code IN (
    'user:read', 'account:read',
    'trade:view', 'trade:analysis',
    'admin:logs'
)
ON CONFLICT (role, permission_id) DO NOTHING;

-- 为 users 表添加 role 索引（如果不存在）
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);
