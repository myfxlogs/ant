-- Migration 149: Revert ON DELETE CASCADE on wallet_transactions.user_id FK
BEGIN;

ALTER TABLE wallet_transactions
  DROP CONSTRAINT wallet_transactions_user_id_fkey;

ALTER TABLE wallet_transactions
  ADD CONSTRAINT wallet_transactions_user_id_fkey
    FOREIGN KEY (user_id) REFERENCES users(id);

COMMIT;
