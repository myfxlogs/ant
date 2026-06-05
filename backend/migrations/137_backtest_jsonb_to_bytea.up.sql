-- 137_backtest_cost_snapshot_jsonb_to_bytea: fix remaining JSONB → BYTEA
-- cost_model_snapshot stores proto binary but was created as JSONB.
-- parameter_overrides/metrics/equity_curve/parameter_space already handled
-- by migrations 134 and 136.

ALTER TABLE backtest_runs ALTER COLUMN cost_model_snapshot TYPE BYTEA USING cost_model_snapshot::text::bytea;
ALTER TABLE backtest_datasets ALTER COLUMN cost_model_snapshot TYPE BYTEA USING cost_model_snapshot::text::bytea;
