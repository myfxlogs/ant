-- 155_system_strategies_schema.down.sql

-- 移除 flag 列
ALTER TABLE strategy_templates DROP COLUMN IF EXISTS flagged_at;
ALTER TABLE strategy_templates DROP COLUMN IF EXISTS flagged_by;
ALTER TABLE strategy_templates DROP COLUMN IF EXISTS flag_reason;
ALTER TABLE strategy_templates DROP COLUMN IF EXISTS flag;

-- 移除 CHECK 约束
ALTER TABLE strategy_templates DROP CONSTRAINT IF EXISTS ck_system_user_id_null;

-- 恢复 user_id NOT NULL（如果还有 NULL 行则先清理）
DELETE FROM strategy_templates WHERE user_id IS NULL;

ALTER TABLE strategy_templates ALTER COLUMN user_id SET NOT NULL;

-- 重新创建 FK（与 migration 012 一致）
ALTER TABLE strategy_templates DROP CONSTRAINT IF EXISTS strategy_templates_user_id_fkey;
ALTER TABLE strategy_templates ADD CONSTRAINT strategy_templates_user_id_fkey
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
