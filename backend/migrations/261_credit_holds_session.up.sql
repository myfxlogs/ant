-- CREDIT-1: Persist credit hold session IDs for crash recovery.
-- Allows CreditService to restore in-memory holds map after restart
-- by querying unsettled ai_hold transactions.

ALTER TABLE credit_transactions
    ADD COLUMN IF NOT EXISTS session_id VARCHAR(255);

CREATE INDEX IF NOT EXISTS idx_credit_transactions_session
    ON credit_transactions(session_id) WHERE session_id IS NOT NULL;
