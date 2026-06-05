-- 138: Add user_id to account_balance_history for data ownership.
-- Backfill from mt_accounts, set orphaned rows to a sentinel, enforce NOT NULL.

ALTER TABLE account_balance_history ADD COLUMN IF NOT EXISTS user_id uuid;

-- Backfill: derive user_id from the account's owner.
UPDATE account_balance_history abh
   SET user_id = ma.user_id
  FROM mt_accounts ma
 WHERE ma.id = abh.account_id
   AND abh.user_id IS NULL;

-- Orphaned rows (account no longer exists) get uuid_nil sentinel.
UPDATE account_balance_history
   SET user_id = '00000000-0000-0000-0000-000000000000'
 WHERE user_id IS NULL;

ALTER TABLE account_balance_history ALTER COLUMN user_id SET NOT NULL;

ALTER TABLE account_balance_history
  ADD CONSTRAINT IF NOT EXISTS account_balance_history_user_id_fkey
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS idx_abh_user_time
    ON account_balance_history (user_id, recorded_at DESC);
