-- 189 down: remove soft-delete infrastructure.

DROP INDEX IF EXISTS idx_mt_accounts_deleted;
ALTER TABLE mt_accounts DROP COLUMN IF EXISTS deleted_at;
