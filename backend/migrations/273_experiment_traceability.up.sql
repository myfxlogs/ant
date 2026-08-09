-- Add traceability columns to strategy_experiments.
-- strategy_name: display name for the experiment (from backtest run name or template name).
-- backtest_run_id: links experiment to its originating backtest run for config inheritance.
ALTER TABLE strategy_experiments
  ADD COLUMN IF NOT EXISTS strategy_name   TEXT DEFAULT '',
  ADD COLUMN IF NOT EXISTS backtest_run_id UUID;
