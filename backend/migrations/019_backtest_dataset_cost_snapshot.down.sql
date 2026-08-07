-- 019_backtest_dataset_cost_snapshot.down.sql
ALTER TABLE backtest_datasets DROP COLUMN IF EXISTS cost_model_snapshot;
