-- 195_subscription_plans.up.sql
-- P3.1: Platform subscription plans + user platform subscriptions.
-- Separate from marketplace user_subscriptions (which track strategy purchases).

CREATE TABLE IF NOT EXISTS subscription_plans (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name            VARCHAR(32) NOT NULL UNIQUE,          -- free / pro / enterprise
    display_name    VARCHAR(64) NOT NULL,
    price_monthly   NUMERIC(10,2) NOT NULL DEFAULT 0,
    price_yearly    NUMERIC(10,2) NOT NULL DEFAULT 0,
    max_ai_tokens_monthly   INTEGER NOT NULL DEFAULT 0,   -- 0 = unlimited
    max_strategies          INTEGER NOT NULL DEFAULT 0,
    max_backtests_daily     INTEGER NOT NULL DEFAULT 0,
    max_live_strategies     INTEGER NOT NULL DEFAULT 0,
    max_symbols_per_strategy INTEGER NOT NULL DEFAULT 0,
    capability_tier         INTEGER NOT NULL DEFAULT 0,   -- maps to risksvc.CapabilityTier 0-3
    features        JSONB NOT NULL DEFAULT '{}'::jsonb,   -- feature flags
    sort_order      INTEGER NOT NULL DEFAULT 0,
    is_active       BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS user_platform_subscriptions (
    id                      UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id                 UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    plan_id                 UUID NOT NULL REFERENCES subscription_plans(id),
    status                  VARCHAR(20) NOT NULL DEFAULT 'active',  -- active / expired / cancelled
    billing_cycle           VARCHAR(10) NOT NULL DEFAULT 'monthly', -- monthly / yearly
    current_period_start    TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    current_period_end      TIMESTAMPTZ NOT NULL,
    auto_renew              BOOLEAN NOT NULL DEFAULT false,
    cancelled_at            TIMESTAMPTZ,
    wallet_transaction_id   UUID REFERENCES wallet_transactions(id) ON DELETE SET NULL, -- link to wallet_transactions
    created_at              TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_user_platform_subs_user ON user_platform_subscriptions(user_id);
CREATE INDEX IF NOT EXISTS idx_user_platform_subs_status ON user_platform_subscriptions(status);
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_platform_subs_active_user
    ON user_platform_subscriptions(user_id) WHERE status = 'active';

-- Seed default plans
INSERT INTO subscription_plans (name, display_name, price_monthly, price_yearly,
    max_ai_tokens_monthly, max_strategies, max_backtests_daily, max_live_strategies,
    max_symbols_per_strategy, capability_tier, features, sort_order)
VALUES
    ('free', 'Free', 0, 0,
     10000, 3, 5, 0, 1, 0,
     '{"multi_symbol": false, "version_history": false, "ai_copilot": true, "marketplace": true}'::jsonb,
     0),
    ('pro', 'Pro', 19.90, 199.00,
     200000, 50, 50, 5, 5, 2,
     '{"multi_symbol": true, "version_history": true, "ai_copilot": true, "marketplace": true}'::jsonb,
     1),
    ('enterprise', 'Enterprise', 99.00, 999.00,
     0, 0, 0, 0, 0, 3,
     '{"multi_symbol": true, "version_history": true, "ai_copilot": true, "marketplace": true, "priority_support": true}'::jsonb,
     2)
ON CONFLICT (name) DO NOTHING;

COMMENT ON TABLE subscription_plans IS 'P3.1: Platform subscription plan definitions (Free/Pro/Enterprise).';
COMMENT ON TABLE user_platform_subscriptions IS 'P3.1: User active platform subscription tracking.';
COMMENT ON COLUMN subscription_plans.capability_tier IS 'Maps to risksvc.CapabilityTier: 0=ViewOnly, 1=Paper, 2=LiveLimited, 3=LiveFull.';
COMMENT ON COLUMN subscription_plans.max_ai_tokens_monthly IS 'Monthly AI token quota; 0 = unlimited.';
