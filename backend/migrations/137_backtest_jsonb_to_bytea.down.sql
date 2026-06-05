-- 137_backtest_cost_snapshot_bytea_to_jsonb: revert BYTEA → JSONB
ALTER TABLE backtest_runs ALTER COLUMN cost_model_snapshot TYPE JSONB USING cost_model_snapshot::text::jsonb;
ALTER TABLE backtest_datasets ALTER COLUMN cost_model_snapshot TYPE JSONB USING cost_model_snapshot::text::jsonb;
