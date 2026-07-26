-- 242: Add bundle_id to user_subscriptions for exact settlement lookup.
-- When a bundle is purchased, each subscription row records the bundle_id.
-- This enables refund logic to find the settlement by exact bundle_id match
-- instead of relying on fragile idempotency_key prefix matching.
ALTER TABLE user_subscriptions
    ADD COLUMN IF NOT EXISTS bundle_id UUID;

CREATE INDEX IF NOT EXISTS idx_subscriptions_bundle ON user_subscriptions(bundle_id) WHERE bundle_id IS NOT NULL;
