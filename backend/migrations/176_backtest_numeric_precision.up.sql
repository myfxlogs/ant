-- 176_backtest_numeric_precision.up.sql
-- Convert backtest financial columns from DOUBLE PRECISION to NUMERIC(20,8)
-- to eliminate float64 precision loss in price/volume/monetary fields.

-- backtest_run_trades: price/volume/monetary fields
ALTER TABLE backtest_run_trades
  ALTER COLUMN volume      TYPE NUMERIC(20,8) USING volume::NUMERIC(20,8),
  ALTER COLUMN open_price  TYPE NUMERIC(20,8) USING open_price::NUMERIC(20,8),
  ALTER COLUMN close_price TYPE NUMERIC(20,8) USING close_price::NUMERIC(20,8),
  ALTER COLUMN pnl         TYPE NUMERIC(20,8) USING pnl::NUMERIC(20,8),
  ALTER COLUMN commission  TYPE NUMERIC(20,8) USING commission::NUMERIC(20,8);

-- backtest_runs: execution config fields (added by migrations 023 and 140)
ALTER TABLE backtest_runs
  ALTER COLUMN initial_capital TYPE NUMERIC(20,8) USING initial_capital::NUMERIC(20,8),
  ALTER COLUMN commission      TYPE NUMERIC(20,8) USING commission::NUMERIC(20,8),
  ALTER COLUMN slippage        TYPE NUMERIC(20,8) USING slippage::NUMERIC(20,8),
  ALTER COLUMN leverage        TYPE NUMERIC(20,8) USING leverage::NUMERIC(20,8);
