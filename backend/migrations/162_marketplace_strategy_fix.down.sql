-- 162_marketplace_strategy_fix.down.sql
-- Revert FK changes and drop added columns.

-- 1. Revert marketplace_strategies FK back to platform_strategies
ALTER TABLE marketplace_strategies DROP CONSTRAINT IF EXISTS marketplace_strategies_strategy_id_fkey;
ALTER TABLE marketplace_strategies
  ADD CONSTRAINT marketplace_strategies_strategy_id_fkey
  FOREIGN KEY (strategy_id) REFERENCES platform_strategies(id) ON DELETE CASCADE;

-- 2. Drop added columns
ALTER TABLE marketplace_strategies DROP COLUMN IF EXISTS code_snippet;
ALTER TABLE marketplace_strategies DROP COLUMN IF EXISTS backtest_snapshot;

-- 3. Revert user_strategy_publishes FK back to platform_strategies
ALTER TABLE user_strategy_publishes DROP CONSTRAINT IF EXISTS user_strategy_publishes_strategy_id_fkey;
ALTER TABLE user_strategy_publishes
  ADD CONSTRAINT user_strategy_publishes_platform_strategy_id_fkey
  FOREIGN KEY (platform_strategy_id) REFERENCES platform_strategies(id) ON DELETE CASCADE;
