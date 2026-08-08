-- LEAKAGE-1 Phase 1: Account binding enforcement.
-- Tracks which MT accounts are bound to a user's platform subscription.
-- The number of bound accounts is limited by subscription_plans.max_mt_accounts (migration 249: free=1, pro=5, enterprise=0/unlimited).

CREATE TABLE IF NOT EXISTS subscription_bound_accounts (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    mt_account_id UUID NOT NULL REFERENCES mt_accounts(id) ON DELETE CASCADE,
    bound_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, mt_account_id)
);

-- Backfill: auto-bind accounts that already have active schedules.
INSERT INTO subscription_bound_accounts (user_id, mt_account_id)
SELECT DISTINCT s.user_id, s.account_id
FROM strategy_schedules s
WHERE s.is_active = true AND s.account_id IS NOT NULL
ON CONFLICT (user_id, mt_account_id) DO NOTHING;

COMMENT ON TABLE subscription_bound_accounts IS 'LEAKAGE-1: MT accounts bound to user subscriptions for tier-based account limit enforcement.';
