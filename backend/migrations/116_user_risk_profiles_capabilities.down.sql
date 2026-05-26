-- 116_user_risk_profiles_capabilities.down.sql

ALTER TABLE user_risk_profiles
    DROP COLUMN IF EXISTS capability_tier,
    DROP COLUMN IF EXISTS order_types_allowed,
    DROP COLUMN IF EXISTS lot_per_order_max,
    DROP COLUMN IF EXISTS daily_order_max,
    DROP COLUMN IF EXISTS leverage_max;
