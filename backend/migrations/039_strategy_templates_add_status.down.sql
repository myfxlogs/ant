-- 039_strategy_templates_add_status.down.sql
-- Auto-generated rollback for 039_strategy_templates_add_status

-- Drop indexes
DROP INDEX IF EXISTS idx_strategy_templates_status;

-- Drop added columns
ALTER TABLE strategy_templates DROP COLUMN IF EXISTS status;

