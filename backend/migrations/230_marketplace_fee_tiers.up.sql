-- Phase 5.3: Tiered platform fee rates
-- Providers get lower platform fees as they accumulate more sales volume.
CREATE TABLE IF NOT EXISTS marketplace_fee_tiers (
    id              SERIAL PRIMARY KEY,
    tier_name       TEXT NOT NULL UNIQUE,              -- starter | growth | pro | elite
    min_sales_count INT NOT NULL DEFAULT 0,             -- minimum total sales to qualify
    fee_rate        NUMERIC(5,4) NOT NULL DEFAULT 0.10, -- platform commission rate
    sort_order      INT NOT NULL DEFAULT 0,
    enabled         BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Seed default tiers
INSERT INTO marketplace_fee_tiers (tier_name, min_sales_count, fee_rate, sort_order) VALUES
    ('starter', 0,   0.10, 1),
    ('growth',  10,  0.08, 2),
    ('pro',     50,  0.05, 3),
    ('elite',   200, 0.03, 4)
ON CONFLICT (tier_name) DO NOTHING;
