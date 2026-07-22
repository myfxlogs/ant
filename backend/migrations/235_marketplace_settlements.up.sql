-- 235: Marketplace freeze settlement mechanism (Phase 5.4).
-- Purchases no longer credit publisher/platform immediately. Instead, a
-- settlement row is created in 'frozen' state. After the refund window
-- (default 7 days), a lazy settlement credits publisher + platform.
-- Refunds within the window simply mark the settlement as 'refunded' and
-- credit the buyer back — publisher wallet is never touched.

CREATE TABLE marketplace_settlements (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    purchase_id        UUID NOT NULL,          -- user_subscriptions.id (no FK to avoid migration coupling)
    buyer_id           UUID NOT NULL,
    provider_id        UUID NOT NULL,
    amount             NUMERIC(20,8) NOT NULL,     -- total amount charged to buyer
    platform_fee       NUMERIC(20,8) NOT NULL,     -- platform fee portion
    provider_amount    NUMERIC(20,8) NOT NULL,     -- provider net portion (amount - platform_fee)
    status             VARCHAR(20) NOT NULL DEFAULT 'frozen',  -- frozen / settled / refunded
    refund_window_days INT NOT NULL DEFAULT 7,
    freezes_at         TIMESTAMPTZ NOT NULL,        -- when freeze starts (= purchase time)
    settles_at         TIMESTAMPTZ NOT NULL,        -- when settlement becomes eligible (= freezes_at + refund_window_days)
    settled_at         TIMESTAMPTZ,
    refunded_at        TIMESTAMPTZ,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_settlements_provider_status ON marketplace_settlements(provider_id, status);
CREATE INDEX idx_settlements_settles_at ON marketplace_settlements(settles_at) WHERE status = 'frozen';
CREATE INDEX idx_settlements_purchase ON marketplace_settlements(purchase_id);
