-- 170_subscription_expiry.up.sql
-- Add expires_at to user_subscriptions for time-limited subscription purchases.

ALTER TABLE user_subscriptions ADD COLUMN IF NOT EXISTS expires_at TIMESTAMPTZ;

COMMENT ON COLUMN user_subscriptions.expires_at IS 'When the subscription expires. NULL = permanent (one-time purchase).';
