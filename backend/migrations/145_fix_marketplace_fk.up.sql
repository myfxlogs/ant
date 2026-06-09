-- 145: fix marketplace_strategies.strategy_id FK → platform_strategies(id).
-- The Publish flow inserts a platform_strategies.id, not a strategies.id.
-- Drop the old FK and add the correct one.
ALTER TABLE marketplace_strategies
    DROP CONSTRAINT IF EXISTS marketplace_strategies_strategy_id_fkey;

ALTER TABLE marketplace_strategies
    ADD CONSTRAINT marketplace_strategies_strategy_id_fkey
        FOREIGN KEY (strategy_id) REFERENCES platform_strategies(id) ON DELETE CASCADE;
