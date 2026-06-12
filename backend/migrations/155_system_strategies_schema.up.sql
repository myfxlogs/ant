-- 155_system_strategies_schema.up.sql
-- 允许 strategy_templates.user_id 为 NULL（平台共享策略不归属任何用户）

-- ============================================
-- 放松 user_id FK 约束
-- ============================================

-- 移除 inline FK（PostgreSQL 自动命名 strategy_templates_user_id_fkey）
ALTER TABLE strategy_templates DROP CONSTRAINT IF EXISTS strategy_templates_user_id_fkey;

-- 允许系统策略不归属任何用户
ALTER TABLE strategy_templates ALTER COLUMN user_id DROP NOT NULL;

-- 重新添加 FK（NULL 值不会被检查）
ALTER TABLE strategy_templates ADD CONSTRAINT strategy_templates_user_id_fkey
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

-- CHECK 约束：系统策略必须 user_id IS NULL
ALTER TABLE strategy_templates ADD CONSTRAINT ck_system_user_id_null
    CHECK (is_system = false OR user_id IS NULL);

COMMENT ON CONSTRAINT ck_system_user_id_null ON strategy_templates IS
    '系统策略 (is_system=true) 的 user_id 必须为 NULL — 平台共享资产，不属于任何用户';

-- 添加 flag 相关列（合规操作）
ALTER TABLE strategy_templates ADD COLUMN IF NOT EXISTS flag VARCHAR(20) NOT NULL DEFAULT '';
ALTER TABLE strategy_templates ADD COLUMN IF NOT EXISTS flag_reason TEXT NOT NULL DEFAULT '';
ALTER TABLE strategy_templates ADD COLUMN IF NOT EXISTS flagged_by UUID;
ALTER TABLE strategy_templates ADD COLUMN IF NOT EXISTS flagged_at TIMESTAMPTZ;

COMMENT ON COLUMN strategy_templates.flag IS '合规标记: "" | flagged | disabled | archived';
COMMENT ON COLUMN strategy_templates.flag_reason IS '合规标记原因';
COMMENT ON COLUMN strategy_templates.flagged_by IS '执行标记的 admin user_id';
COMMENT ON COLUMN strategy_templates.flagged_at IS '标记时间';

-- 添加 i18n 列（Migration 050 已添加，这里是幂等保护）
-- ALTER TABLE strategy_templates ADD COLUMN IF NOT EXISTS i18n JSONB DEFAULT '{}'::JSONB;
