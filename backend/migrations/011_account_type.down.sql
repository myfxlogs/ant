-- 011_account_type.down.sql
-- Auto-generated rollback for 011_account_type

-- Drop added columns
ALTER TABLE mt_accounts DROP COLUMN IF EXISTS account_type;

