-- 223_notified_expiring.down.sql
ALTER TABLE marketplace_trials DROP COLUMN IF EXISTS notified_expiring;
ALTER TABLE user_subscriptions DROP COLUMN IF EXISTS notified_expiring;
