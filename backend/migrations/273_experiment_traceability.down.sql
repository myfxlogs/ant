ALTER TABLE strategy_experiments
  DROP COLUMN IF EXISTS strategy_name,
  DROP COLUMN IF EXISTS backtest_run_id;
