-- 169_purchase_idempotency.down.sql

DROP INDEX IF EXISTS idx_subscriptions_idempotency;
ALTER TABLE user_subscriptions DROP COLUMN IF EXISTS idempotency_key;
