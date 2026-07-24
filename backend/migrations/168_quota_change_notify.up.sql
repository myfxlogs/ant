-- 168_quota_change_notify.up.sql
-- NOTIFY trigger for user_platform_subscriptions changes.
-- Enables QuotaChecker to refresh its in-memory cache via PG LISTEN
-- instead of polling with a ticker.

CREATE OR REPLACE FUNCTION notify_quota_change() RETURNS trigger AS $$
BEGIN
    PERFORM pg_notify('quota_change', '');
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_quota_change_insert ON user_platform_subscriptions;
DROP TRIGGER IF EXISTS trg_quota_change_update ON user_platform_subscriptions;
DROP TRIGGER IF EXISTS trg_quota_change_delete ON user_platform_subscriptions;

CREATE TRIGGER trg_quota_change_insert
    AFTER INSERT ON user_platform_subscriptions
    FOR EACH ROW EXECUTE FUNCTION notify_quota_change();

CREATE TRIGGER trg_quota_change_update
    AFTER UPDATE ON user_platform_subscriptions
    FOR EACH ROW EXECUTE FUNCTION notify_quota_change();

CREATE TRIGGER trg_quota_change_delete
    AFTER DELETE ON user_platform_subscriptions
    FOR EACH ROW EXECUTE FUNCTION notify_quota_change();
