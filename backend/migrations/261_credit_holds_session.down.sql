ALTER TABLE credit_transactions DROP COLUMN IF EXISTS session_id;
DROP INDEX IF EXISTS idx_credit_transactions_session;
