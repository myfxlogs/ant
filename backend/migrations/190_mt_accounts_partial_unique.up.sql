-- 190: Replace full unique constraint with partial index (WHERE deleted_at IS NULL).
-- Migration 189 added soft-delete (deleted_at), but migration 115's unique constraint
-- uk_mt_account_login still enforced uniqueness across ALL rows including soft-deleted ones.
-- This prevented users from re-binding an MT account they had previously deleted.
-- Fix: partial unique index only enforces uniqueness among active (non-deleted) accounts.

ALTER TABLE mt_accounts DROP CONSTRAINT IF EXISTS uk_mt_account_login;
CREATE UNIQUE INDEX uk_mt_account_login_active
  ON mt_accounts (login, mt_type, broker_server)
  WHERE deleted_at IS NULL;
