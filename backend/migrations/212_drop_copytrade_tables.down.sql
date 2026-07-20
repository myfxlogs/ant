-- 212_drop_copytrade_tables.down.sql
-- Recreate copytrade tables (for rollback only).

CREATE TABLE IF NOT EXISTS copytrade_signals (
    signal_id   VARCHAR(128) PRIMARY KEY,
    strategy_id UUID NOT NULL,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at  TIMESTAMPTZ NOT NULL DEFAULT (now() + INTERVAL '24 hours')
);

CREATE INDEX IF NOT EXISTS idx_copytrade_signals_expires ON copytrade_signals(expires_at);

CREATE TABLE IF NOT EXISTS copy_trade_links (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subscription_id     UUID NOT NULL REFERENCES user_subscriptions(id),
    from_account_id     UUID NOT NULL,
    to_account_id       UUID NOT NULL,
    ratio               DOUBLE PRECISION NOT NULL DEFAULT 1.0,
    max_lots            DOUBLE PRECISION NOT NULL DEFAULT 10.0,
    active              BOOLEAN NOT NULL DEFAULT true,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);
