-- 099_broker_symbols.down.sql
-- Auto-generated rollback for 099_broker_symbols

-- Drop indexes
DROP INDEX IF EXISTS idx_broker_symbols_canonical;

-- Drop tables
DROP TABLE IF EXISTS broker_symbols CASCADE;

