-- 191: Drop strategy_execution_logs — dead table.
-- Writes were never wired; reads always returned empty.
-- Schedule health stats now served by schedule_run_logs.
DROP TABLE IF EXISTS strategy_execution_logs;
