-- 032_strategy_schedules_add_enable_count.down.sql
-- Auto-generated rollback for 032_strategy_schedules_add_enable_count

-- Drop indexes
DROP INDEX IF EXISTS idx_strategy_schedules_enable_count;

-- Drop added columns
ALTER TABLE strategy_schedules DROP COLUMN IF EXISTS enable_count;

