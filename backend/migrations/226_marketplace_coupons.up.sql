-- 226_marketplace_coupons.up.sql
CREATE TABLE IF NOT EXISTS marketplace_coupons (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(50) UNIQUE NOT NULL,
    discount_type VARCHAR(10) NOT NULL,       -- percentage / fixed
    discount_value NUMERIC(20,8) NOT NULL,    -- 0-100 (percentage) or amount (fixed)
    min_purchase_amount NUMERIC(20,8) DEFAULT 0,
    max_uses INT DEFAULT 0,                   -- 0 = unlimited
    used_count INT DEFAULT 0,
    expires_at TIMESTAMPTZ,                   -- NULL = never expires
    applicable_strategy_ids UUID[] DEFAULT '{}',
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    enabled BOOLEAN DEFAULT true
);
CREATE INDEX IF NOT EXISTS idx_coupons_code ON marketplace_coupons(code) WHERE enabled = true;
