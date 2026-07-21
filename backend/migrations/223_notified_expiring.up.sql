-- 223_notified_expiring.up.sql
-- Add notified_expiring flag to prevent duplicate expiration notifications.

ALTER TABLE marketplace_trials ADD COLUMN IF NOT EXISTS notified_expiring BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE user_subscriptions ADD COLUMN IF NOT EXISTS notified_expiring BOOLEAN NOT NULL DEFAULT false;
