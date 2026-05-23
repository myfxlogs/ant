-- 019_backtest_dataset_cost_snapshot.up.sql
-- Store cost model snapshot together with frozen backtest dataset for reproducibility

ALTER TABLE backtest_datasets
    ADD COLUMN IF NOT EXISTS cost_model_snapshot JSONB;
