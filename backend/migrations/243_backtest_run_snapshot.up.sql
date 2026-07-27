-- Add backtest_snapshot column to store server-generated BacktestSnapshot proto.
-- Populated when a backtest run succeeds; read by marketplace publish quality gate.
ALTER TABLE backtest_runs ADD COLUMN IF NOT EXISTS backtest_snapshot BYTEA;
