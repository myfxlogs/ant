-- Migration 233: Composite index for fee tier sales count queries.
-- getEffectiveFeeRateTx runs COUNT(*) WHERE user_id = $1 AND tx_type = $2
-- on every purchase. Without this index, it scans all wallet_transactions
-- for that user. The composite index makes it an index-only scan.

CREATE INDEX IF NOT EXISTS idx_wallet_transactions_user_tx_type
  ON wallet_transactions (user_id, tx_type);
