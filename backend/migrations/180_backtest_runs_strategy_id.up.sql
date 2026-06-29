-- 180_backtest_runs_strategy_id.up.sql
-- Link backtest_runs to imported_strategies for source-of-truth traceability.
-- When strategy_id is set, strategy_code is populated from imported_strategies
-- at run creation time (denormalized for async worker access).

ALTER TABLE backtest_runs
  ADD COLUMN IF NOT EXISTS strategy_id UUID REFERENCES imported_strategies(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_backtest_runs_strategy_id ON backtest_runs(strategy_id) WHERE strategy_id IS NOT NULL;
