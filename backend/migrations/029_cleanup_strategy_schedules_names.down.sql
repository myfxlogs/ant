-- 029_cleanup_strategy_schedules_names.down.sql
-- Auto-generated rollback for 029_cleanup_strategy_schedules_names

-- Drop triggers
DROP TRIGGER IF EXISTS update_strategy_schedules_updated_at ON strategy_schedules;

