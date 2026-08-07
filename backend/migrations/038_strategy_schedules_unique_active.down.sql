-- 038_strategy_schedules_unique_active.down.sql
-- Auto-generated rollback for 038_strategy_schedules_unique_active

-- Drop indexes
DROP INDEX IF EXISTS ux_strategy_schedules_active_unique;

