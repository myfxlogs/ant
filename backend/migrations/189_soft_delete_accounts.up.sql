-- 189_soft_delete_accounts: soft-delete support for mt_accounts
-- Date: 2026-07-06

ALTER TABLE mt_accounts ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;
CREATE INDEX IF NOT EXISTS idx_mt_accounts_deleted ON mt_accounts (deleted_at) WHERE deleted_at IS NOT NULL;
