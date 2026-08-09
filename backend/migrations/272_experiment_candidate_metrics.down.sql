ALTER TABLE strategy_experiment_candidates
  DROP COLUMN IF EXISTS total_return,
  DROP COLUMN IF EXISTS annual_return,
  DROP COLUMN IF EXISTS sharpe_ratio,
  DROP COLUMN IF EXISTS max_drawdown,
  DROP COLUMN IF EXISTS win_rate,
  DROP COLUMN IF EXISTS profit_factor,
  DROP COLUMN IF EXISTS total_trades;
