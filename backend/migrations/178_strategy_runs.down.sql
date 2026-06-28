-- 178_strategy_runs.down.sql
-- Reverse: drop run_id column + strategy_runs table.

ALTER TABLE strategy_signals DROP COLUMN IF EXISTS run_id;
DROP TABLE IF EXISTS strategy_runs;
