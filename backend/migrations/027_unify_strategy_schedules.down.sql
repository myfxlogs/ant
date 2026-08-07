-- 027_unify_strategy_schedules.down.sql
-- Auto-generated rollback for 027_unify_strategy_schedules

-- Drop triggers
DROP TRIGGER IF EXISTS update_strategy_schedules_updated_at ON strategy_schedules;

-- Drop indexes
DROP INDEX IF EXISTS idx_strategy_schedules_account_id;
DROP INDEX IF EXISTS idx_strategy_schedules_is_active;
DROP INDEX IF EXISTS idx_strategy_schedules_next_run_at;
DROP INDEX IF EXISTS idx_strategy_schedules_risk_level;
DROP INDEX IF EXISTS idx_strategy_schedules_symbol;
DROP INDEX IF EXISTS idx_strategy_schedules_template_id;
DROP INDEX IF EXISTS idx_strategy_schedules_user_id;

