-- 222_marketplace_notification_prefs.up.sql
-- User notification preferences for marketplace events.

CREATE TABLE IF NOT EXISTS marketplace_notification_prefs (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    new_strategy_enabled BOOLEAN NOT NULL DEFAULT true,
    price_change_enabled BOOLEAN NOT NULL DEFAULT true,
    sub_expiring_enabled BOOLEAN NOT NULL DEFAULT true,
    performance_alert_enabled BOOLEAN NOT NULL DEFAULT true,
    new_rating_enabled BOOLEAN NOT NULL DEFAULT true,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
