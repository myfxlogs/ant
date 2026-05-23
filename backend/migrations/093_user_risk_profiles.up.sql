-- 093_user_risk_profiles.up.sql
-- M3-2: user-level risk profile — replaces Strategy.RiskControl JSONB field.
-- Each user has one profile; strategies inherit their owner's profile.

CREATE TABLE IF NOT EXISTS user_risk_profiles (
    user_id              UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    max_positions        INT NOT NULL DEFAULT 5,
    daily_loss_limit     NUMERIC(18,2),              -- absolute value; NULL = disabled
    max_drawdown_pct     NUMERIC(5,2),               -- 0-100; NULL = disabled
    margin_level_min     NUMERIC(5,2) DEFAULT 150.0, -- 150% minimum margin level
    session_enabled      BOOLEAN NOT NULL DEFAULT true,
    slippage_max_points  NUMERIC(8,2),               -- NULL = disabled
    reject_rate_max      NUMERIC(5,2),               -- 0-1; NULL = disabled
    heartbeat_timeout_s  INT DEFAULT 120,            -- seconds; NULL = disabled
    killswitch_enabled   BOOLEAN NOT NULL DEFAULT false,
    symbol_whitelist     TEXT[],                      -- NULL = all symbols allowed
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now()
);

COMMENT ON TABLE user_risk_profiles IS 'Per-user risk profile — strategies inherit owner profile. Replaces Strategy.RiskControl JSONB';
