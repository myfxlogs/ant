-- 005_strategies.down.sql
-- Auto-generated rollback for 005_strategies

-- Drop triggers
DROP TRIGGER IF EXISTS update_strategies_updated_at ON strategies;

-- Drop indexes
DROP INDEX IF EXISTS idx_strategies_account_id;
DROP INDEX IF EXISTS idx_strategies_status;
DROP INDEX IF EXISTS idx_strategies_symbol;
DROP INDEX IF EXISTS idx_strategies_user_id;
DROP INDEX IF EXISTS idx_strategy_signals_account_id;
DROP INDEX IF EXISTS idx_strategy_signals_created_at;
DROP INDEX IF EXISTS idx_strategy_signals_status;
DROP INDEX IF EXISTS idx_strategy_signals_strategy_id;

-- Drop tables
DROP TABLE IF EXISTS strategy_signals CASCADE;
DROP TABLE IF EXISTS strategies CASCADE;

