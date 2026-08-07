-- 041_backtest_runs_template_ref.down.sql
-- Auto-generated rollback for 041_backtest_runs_template_ref

-- Drop indexes
DROP INDEX IF EXISTS idx_backtest_runs_template_draft_id;
DROP INDEX IF EXISTS idx_backtest_runs_template_id;

