-- 116_user_risk_profiles_capabilities.up.sql
-- M10-BASE-C1: Extend user_risk_profiles with capability tier + order constraints.
-- Adds 5 columns to existing table (migration 093), does NOT create a new table.

ALTER TABLE user_risk_profiles
    ADD COLUMN IF NOT EXISTS capability_tier      INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS order_types_allowed   TEXT[] DEFAULT NULL,
    ADD COLUMN IF NOT EXISTS lot_per_order_max     NUMERIC(18,2) DEFAULT NULL,
    ADD COLUMN IF NOT EXISTS daily_order_max       INT DEFAULT NULL,
    ADD COLUMN IF NOT EXISTS leverage_max          NUMERIC(8,2) DEFAULT NULL;

COMMENT ON COLUMN user_risk_profiles.capability_tier IS 'Tier 0-3: 0=view-only, 1=paper, 2=live-limited, 3=live-full';
COMMENT ON COLUMN user_risk_profiles.order_types_allowed IS 'Allowed order types (MARKET, LIMIT, STOP, STOP_LIMIT); NULL = all allowed';
COMMENT ON COLUMN user_risk_profiles.lot_per_order_max IS 'Maximum lot size per single order; NULL = unlimited';
COMMENT ON COLUMN user_risk_profiles.daily_order_max IS 'Maximum number of orders per day; NULL = unlimited';
COMMENT ON COLUMN user_risk_profiles.leverage_max IS 'Maximum leverage ratio (e.g. 100 = 100:1); NULL = broker default';
