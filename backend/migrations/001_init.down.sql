-- 001_init.down.sql
-- Auto-generated rollback for 001_init

-- Drop triggers
DROP TRIGGER IF EXISTS update_mt_accounts_updated_at ON mt_accounts;
DROP TRIGGER IF EXISTS update_orders_updated_at ON orders;
DROP TRIGGER IF EXISTS update_positions_updated_at ON positions;
DROP TRIGGER IF EXISTS update_symbols_updated_at ON symbols;
DROP TRIGGER IF EXISTS update_users_updated_at ON users;

-- Drop indexes
DROP INDEX IF EXISTS idx_mt_accounts_mt_type;
DROP INDEX IF EXISTS idx_mt_accounts_status;
DROP INDEX IF EXISTS idx_mt_accounts_user;
DROP INDEX IF EXISTS idx_orders_mt_account;
DROP INDEX IF EXISTS idx_orders_platform;
DROP INDEX IF EXISTS idx_orders_symbol;
DROP INDEX IF EXISTS idx_positions_mt_account;
DROP INDEX IF EXISTS idx_positions_platform;
DROP INDEX IF EXISTS idx_positions_symbol;
DROP INDEX IF EXISTS idx_symbols_symbol;
DROP INDEX IF EXISTS idx_users_email;
DROP INDEX IF EXISTS idx_users_status;

-- Drop tables
DROP TABLE IF EXISTS users CASCADE;
DROP TABLE IF EXISTS system_config CASCADE;
DROP TABLE IF EXISTS symbols CASCADE;
DROP TABLE IF EXISTS positions CASCADE;
DROP TABLE IF EXISTS orders CASCADE;
DROP TABLE IF EXISTS mt_accounts CASCADE;

