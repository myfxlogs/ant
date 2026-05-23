-- 095_marketplace_strategies.up.sql
-- M5-1: strategy marketplace — users publish strategies for others to subscribe/purchase.

CREATE TABLE IF NOT EXISTS marketplace_strategies (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    strategy_id     UUID NOT NULL REFERENCES strategies(id) ON DELETE CASCADE,
    publisher_id    UUID NOT NULL REFERENCES users(id),
    title           TEXT NOT NULL,
    description     TEXT NOT NULL,
    price_model     VARCHAR(16) NOT NULL DEFAULT 'free', -- free, monthly, once
    price_amount    NUMERIC(10,2),                        -- NULL for free
    asset_class     TEXT NOT NULL,                        -- forex, crypto, commodity, index
    symbols         TEXT[] NOT NULL DEFAULT '{}',         -- canonical symbols
    timeframe       VARCHAR(8),                           -- M1, M5, M15, H1, H4, D1
    risk_level      VARCHAR(8) NOT NULL DEFAULT 'medium', -- low, medium, high
    tags            TEXT[] DEFAULT '{}',
    total_subscribers INT NOT NULL DEFAULT 0,
    total_pnl       NUMERIC(18,2),
    win_rate        NUMERIC(5,2),
    status          VARCHAR(16) NOT NULL DEFAULT 'published', -- published, hidden, removed
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_marketplace_status ON marketplace_strategies(status);
CREATE INDEX IF NOT EXISTS idx_marketplace_publisher ON marketplace_strategies(publisher_id);
CREATE INDEX IF NOT EXISTS idx_marketplace_asset_class ON marketplace_strategies(asset_class);

COMMENT ON TABLE marketplace_strategies IS 'Strategy marketplace — user-published strategies for community sharing';
