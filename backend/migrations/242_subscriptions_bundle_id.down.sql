DROP INDEX IF EXISTS idx_subscriptions_bundle;
ALTER TABLE user_subscriptions DROP COLUMN IF EXISTS bundle_id;
