-- Migration 151: Add soft-delete support to users table.
-- deleted_at = NULL means active user; NOT NULL means soft-deleted.
-- The hard-delete cron (30-day retention) physically removes expired users,
-- at which point the CASCADE/SET NULL FKs from migrations 149/150 take effect.
BEGIN;

ALTER TABLE users ADD COLUMN deleted_at TIMESTAMPTZ;
CREATE INDEX idx_users_deleted_at ON users (deleted_at);

COMMIT;
