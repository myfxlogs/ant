ALTER TABLE backtest_runs DROP COLUMN IF EXISTS config_snapshot;
ALTER TABLE backtest_runs DROP COLUMN IF EXISTS strict_mode;
ALTER TABLE backtest_runs DROP COLUMN IF EXISTS trade_direction;
ALTER TABLE backtest_runs DROP COLUMN IF EXISTS leverage;
ALTER TABLE backtest_runs DROP COLUMN IF EXISTS slippage;
ALTER TABLE backtest_runs DROP COLUMN IF EXISTS commission;
