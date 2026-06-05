-- 136_backtest_drop_legacy_jsonb: restore metrics/equity_curve JSONB columns
ALTER TABLE backtest_runs ADD COLUMN IF NOT EXISTS metrics JSONB;
ALTER TABLE backtest_runs ADD COLUMN IF NOT EXISTS equity_curve JSONB;
