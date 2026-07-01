-- 181_strategy_templates_strategy_id.down.sql

DROP INDEX IF EXISTS idx_strategy_templates_strategy_id;
ALTER TABLE strategy_templates DROP COLUMN IF EXISTS strategy_id;
