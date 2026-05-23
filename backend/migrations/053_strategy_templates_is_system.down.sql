-- 053_strategy_templates_is_system.down.sql
DROP INDEX IF EXISTS idx_strategy_templates_user_system;
ALTER TABLE strategy_templates DROP COLUMN IF EXISTS is_system;
