-- Migration 149: Add ON DELETE CASCADE to wallet_transactions.user_id FK
-- The original migration 148 missed CASCADE on the users FK, causing
-- user deletion to fail with a foreign key violation when the user
-- has wallet transaction records.
BEGIN;

ALTER TABLE wallet_transactions
  DROP CONSTRAINT wallet_transactions_user_id_fkey;

ALTER TABLE wallet_transactions
  ADD CONSTRAINT wallet_transactions_user_id_fkey
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

COMMIT;
