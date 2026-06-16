-- 160_account_balance_history_numeric.up.sql
-- Fix DOUBLE PRECISION → NUMERIC(20,8) for financial columns.
-- DOUBLE PRECISION introduces IEEE 754 rounding errors in balance/equity/margin/free_margin.
-- NUMERIC(20,8) ensures exact decimal representation (matching project precision rules).

ALTER TABLE account_balance_history
    ALTER COLUMN balance     TYPE NUMERIC(20,8) USING balance::NUMERIC(20,8),
    ALTER COLUMN equity      TYPE NUMERIC(20,8) USING equity::NUMERIC(20,8),
    ALTER COLUMN margin      TYPE NUMERIC(20,8) USING margin::NUMERIC(20,8),
    ALTER COLUMN free_margin TYPE NUMERIC(20,8) USING free_margin::NUMERIC(20,8);

-- backtest_run_trades also uses DOUBLE PRECISION for financial columns
ALTER TABLE backtest_run_trades
    ALTER COLUMN volume      TYPE NUMERIC(20,8) USING volume::NUMERIC(20,8),
    ALTER COLUMN open_price  TYPE NUMERIC(20,8) USING open_price::NUMERIC(20,8),
    ALTER COLUMN close_price TYPE NUMERIC(20,8) USING close_price::NUMERIC(20,8),
    ALTER COLUMN pnl         TYPE NUMERIC(20,8) USING pnl::NUMERIC(20,8),
    ALTER COLUMN commission  TYPE NUMERIC(20,8) USING commission::NUMERIC(20,8);
