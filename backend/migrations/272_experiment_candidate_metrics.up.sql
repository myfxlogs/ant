-- Add raw backtest metric columns to strategy_experiment_candidates.
-- These store the original (unscored) metrics from in-process experiment backtests,
-- eliminating the need for full backtest_runs records for experiment candidates.
ALTER TABLE strategy_experiment_candidates
  ADD COLUMN IF NOT EXISTS total_return   DOUBLE PRECISION,
  ADD COLUMN IF NOT EXISTS annual_return  DOUBLE PRECISION,
  ADD COLUMN IF NOT EXISTS sharpe_ratio   DOUBLE PRECISION,
  ADD COLUMN IF NOT EXISTS max_drawdown   DOUBLE PRECISION,
  ADD COLUMN IF NOT EXISTS win_rate       DOUBLE PRECISION,
  ADD COLUMN IF NOT EXISTS profit_factor  DOUBLE PRECISION,
  ADD COLUMN IF NOT EXISTS total_trades   INTEGER;
