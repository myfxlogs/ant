-- 169_purchase_idempotency.up.sql
-- Add idempotency_key to user_subscriptions for safe purchase retries.

ALTER TABLE user_subscriptions ADD COLUMN IF NOT EXISTS idempotency_key VARCHAR(128);
CREATE UNIQUE INDEX IF NOT EXISTS idx_subscriptions_idempotency ON user_subscriptions(idempotency_key) WHERE idempotency_key IS NOT NULL;

COMMENT ON COLUMN user_subscriptions.idempotency_key IS 'Client-generated unique key to prevent duplicate purchases on network retry.';
