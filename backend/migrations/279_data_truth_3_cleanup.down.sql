-- 279_data_truth_3_cleanup.down.sql
ALTER TABLE mt_accounts ADD COLUMN IF NOT EXISTS last_checked_at TIMESTAMP;
COMMENT ON VIEW mt_accounts_v2 IS NULL;
