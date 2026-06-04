-- 132: Add parameter_overrides column to backtest_runs for Smart Tuning (Phase 2)
ALTER TABLE backtest_runs ADD COLUMN IF NOT EXISTS parameter_overrides JSONB DEFAULT '{}';
