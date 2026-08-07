-- 017_backtest_datasets.down.sql
-- Auto-generated rollback for 017_backtest_datasets

-- Drop indexes
DROP INDEX IF EXISTS idx_backtest_dataset_bars_dataset_time;
DROP INDEX IF EXISTS idx_backtest_datasets_account;
DROP INDEX IF EXISTS idx_backtest_datasets_symbol_tf;
DROP INDEX IF EXISTS idx_backtest_datasets_user;

-- Drop tables
DROP TABLE IF EXISTS backtest_datasets CASCADE;
DROP TABLE IF EXISTS backtest_dataset_bars CASCADE;

