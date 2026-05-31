-- 128_account_indexes down migration

DROP TRIGGER IF EXISTS trg_mt_accounts_updated_at ON mt_accounts;
DROP FUNCTION IF EXISTS update_updated_at_column();

DROP INDEX IF EXISTS idx_account_balance_history_lookup;
DROP INDEX IF EXISTS idx_trade_records_account_close;
DROP INDEX IF EXISTS idx_mt_accounts_disabled;
DROP INDEX IF EXISTS idx_mt_accounts_disabled_status;
DROP INDEX IF EXISTS idx_mt_accounts_user_status;
