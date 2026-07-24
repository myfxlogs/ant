-- 236_marketplace_unique_published.up.sql
-- H1: Prevent duplicate marketplace listings for the same strategy.
-- Partial unique index: only one 'published' row per strategy_id at a time.

CREATE UNIQUE INDEX IF NOT EXISTS uq_marketplace_strategies_strategy_id_published
    ON marketplace_strategies (strategy_id)
    WHERE status = 'published';
