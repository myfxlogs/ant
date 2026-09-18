-- 280_mt_accounts_margin_mode.down.sql
ALTER TABLE mt_accounts DROP COLUMN IF EXISTS margin_mode;
