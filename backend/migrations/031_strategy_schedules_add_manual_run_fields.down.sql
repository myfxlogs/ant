-- 031_strategy_schedules_add_manual_run_fields.down.sql
-- Auto-generated rollback for 031_strategy_schedules_add_manual_run_fields

-- Drop indexes
DROP INDEX IF EXISTS idx_strategy_schedules_manual_run_count;

-- Drop added columns
ALTER TABLE strategy_schedules DROP COLUMN IF EXISTS last_manual_error;
ALTER TABLE strategy_schedules DROP COLUMN IF EXISTS last_manual_run_at;
ALTER TABLE strategy_schedules DROP COLUMN IF EXISTS manual_run_count;

