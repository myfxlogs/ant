-- 209_ledger_integrity.down.sql

DROP INDEX IF EXISTS idx_ledger_outbox_unsent;
DROP TABLE IF EXISTS ledger_outbox;

DROP TRIGGER IF EXISTS wt_append_only ON wallet_transactions;
DROP FUNCTION IF EXISTS wt_no_mutate();

DROP INDEX IF EXISTS idx_wallet_tx_idem_key;

ALTER TABLE wallet_transactions
  DROP COLUMN IF EXISTS idem_key,
  DROP COLUMN IF EXISTS entry_hash,
  DROP COLUMN IF EXISTS prev_hash,
  DROP COLUMN IF EXISTS seq;

ALTER TABLE user_wallets DROP CONSTRAINT IF EXISTS chk_balance_nonneg;
