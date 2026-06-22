-- 170_subscription_expiry.down.sql

ALTER TABLE user_subscriptions DROP COLUMN IF EXISTS expires_at;
