-- 089_strategy_symbols.up.sql
-- M1 strategy <→ canonical symbol 关联表 (user-scoped whitelist)
-- 取代 strategies.symbol 裸字符串字段

CREATE TABLE IF NOT EXISTS strategy_symbols (
    strategy_id  uuid NOT NULL REFERENCES strategies(id) ON DELETE CASCADE,
    canonical    text NOT NULL REFERENCES canonical_symbols(canonical),
    enabled      boolean NOT NULL DEFAULT true,
    created_at   timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (strategy_id, canonical)
);

CREATE INDEX IF NOT EXISTS idx_strategy_symbols_canonical ON strategy_symbols(canonical);

COMMENT ON TABLE strategy_symbols IS 'Strategy whitelist — which canonical symbols a strategy trades on';
