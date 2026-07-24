-- 168_quota_change_notify.down.sql
DROP TRIGGER IF EXISTS trg_quota_change_insert ON user_platform_subscriptions;
DROP TRIGGER IF EXISTS trg_quota_change_update ON user_platform_subscriptions;
DROP TRIGGER IF EXISTS trg_quota_change_delete ON user_platform_subscriptions;
DROP FUNCTION IF EXISTS notify_quota_change();
