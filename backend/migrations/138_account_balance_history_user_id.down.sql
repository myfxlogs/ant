ALTER TABLE account_balance_history DROP CONSTRAINT IF EXISTS account_balance_history_user_id_fkey;
DROP INDEX IF EXISTS idx_abh_user_time;
ALTER TABLE account_balance_history DROP COLUMN IF EXISTS user_id;
