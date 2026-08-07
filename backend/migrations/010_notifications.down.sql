-- 010_notifications.down.sql
-- Auto-generated rollback for 010_notifications

-- Drop indexes
DROP INDEX IF EXISTS idx_notifications_user_unread;

-- Drop tables
DROP TABLE IF EXISTS notifications CASCADE;

