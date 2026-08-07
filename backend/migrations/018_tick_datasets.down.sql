-- 018_tick_datasets.down.sql
-- Auto-generated rollback for 018_tick_datasets

-- Drop indexes
DROP INDEX IF EXISTS idx_tick_dataset_ticks_dataset_time;
DROP INDEX IF EXISTS idx_tick_datasets_account;
DROP INDEX IF EXISTS idx_tick_datasets_symbol;
DROP INDEX IF EXISTS idx_tick_datasets_user;

-- Drop tables
DROP TABLE IF EXISTS tick_datasets CASCADE;
DROP TABLE IF EXISTS tick_dataset_ticks CASCADE;

