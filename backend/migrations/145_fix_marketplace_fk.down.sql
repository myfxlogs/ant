-- 145 down: revert FK back to strategies(id).
ALTER TABLE marketplace_strategies
    DROP CONSTRAINT IF EXISTS marketplace_strategies_strategy_id_fkey;

ALTER TABLE marketplace_strategies
    ADD CONSTRAINT marketplace_strategies_strategy_id_fkey
        FOREIGN KEY (strategy_id) REFERENCES strategies(id) ON DELETE CASCADE;
