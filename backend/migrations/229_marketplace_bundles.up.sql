-- Phase 5.2: Strategy Bundles
CREATE TABLE IF NOT EXISTS marketplace_bundles (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title           TEXT NOT NULL,
    description     TEXT NOT NULL DEFAULT '',
    publisher_id    UUID NOT NULL,
    price_model     TEXT NOT NULL DEFAULT 'once',     -- once | subscription
    price_amount    NUMERIC(20,8) NOT NULL DEFAULT 0,
    platform_fee_rate NUMERIC(5,4) NOT NULL DEFAULT 0.10,
    status          TEXT NOT NULL DEFAULT 'published', -- published | hidden
    total_purchases INT NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS marketplace_bundle_items (
    bundle_id       UUID NOT NULL REFERENCES marketplace_bundles(id) ON DELETE CASCADE,
    strategy_id     UUID NOT NULL,
    sort_order      INT NOT NULL DEFAULT 0,
    PRIMARY KEY (bundle_id, strategy_id)
);

CREATE INDEX idx_bundle_items_strategy ON marketplace_bundle_items(strategy_id);
CREATE INDEX idx_bundles_publisher ON marketplace_bundles(publisher_id);
CREATE INDEX idx_bundles_status ON marketplace_bundles(status);
