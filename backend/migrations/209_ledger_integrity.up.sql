-- 209_ledger_integrity.up.sql
-- ADR-0026 Phase D: Ledger integrity — hash chain, idempotency, append-only, outbox.
-- Red lines R7 (idempotent), R8 (hash chain + append-only), R9 (balance >= 0).

-- 1. Balance non-negative constraint (R9).
ALTER TABLE user_wallets ADD CONSTRAINT chk_balance_nonneg CHECK (balance >= 0);
ALTER TABLE user_wallets ADD CONSTRAINT chk_frozen_nonneg CHECK (frozen_balance >= 0);

-- 2. Hash chain columns on wallet_transactions (R8).
ALTER TABLE wallet_transactions
  ADD COLUMN IF NOT EXISTS seq       BIGINT GENERATED ALWAYS AS IDENTITY,
  ADD COLUMN IF NOT EXISTS prev_hash BYTEA,
  ADD COLUMN IF NOT EXISTS entry_hash BYTEA,
  ADD COLUMN IF NOT EXISTS idem_key  TEXT;

-- Unique index on idem_key for ON CONFLICT idempotency (R7).
-- Partial index: only non-NULL idem_keys are unique-checked.
CREATE UNIQUE INDEX IF NOT EXISTS idx_wallet_tx_idem_key
  ON wallet_transactions(idem_key) WHERE idem_key IS NOT NULL;

-- 3. Append-only trigger: block UPDATE (except entry_hash NULL→non-NULL) and DELETE (R8).
-- entry_hash is set after INSERT because seq is GENERATED ALWAYS AS IDENTITY
-- and is needed for the hash computation — this is the only allowed UPDATE,
-- and only from NULL to a non-NULL value (one-shot, no re-write).
CREATE OR REPLACE FUNCTION wt_no_mutate() RETURNS trigger AS $$
  BEGIN
    IF TG_OP = 'DELETE' THEN
      RAISE EXCEPTION 'wallet_transactions is append-only (R8)';
    END IF;
    -- UPDATE: allow only if entry_hash is being set from NULL to non-NULL
    -- and all other columns are unchanged.
    IF OLD.entry_hash IS NOT NULL THEN
      RAISE EXCEPTION 'wallet_transactions is append-only (R8): entry_hash already set';
    END IF
    IF NEW.entry_hash IS NULL THEN
      RAISE EXCEPTION 'wallet_transactions is append-only (R8): cannot clear entry_hash';
    END IF
    IF NEW.id <> OLD.id
       OR NEW.wallet_id <> OLD.wallet_id
       OR NEW.user_id <> OLD.user_id
       OR NEW.tx_type <> OLD.tx_type
       OR NEW.amount <> OLD.amount
       OR NEW.balance_before <> OLD.balance_before
       OR NEW.balance_after <> OLD.balance_after
       OR NEW.description IS DISTINCT FROM OLD.description
       OR NEW.operator_id IS DISTINCT FROM OLD.operator_id
       OR NEW.created_at <> OLD.created_at
       OR NEW.seq <> OLD.seq
       OR NEW.prev_hash IS DISTINCT FROM OLD.prev_hash
       OR NEW.idem_key IS DISTINCT FROM OLD.idem_key
    THEN
      RAISE EXCEPTION 'wallet_transactions is append-only (R8)';
    END IF;
    RETURN NEW;
  END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS wt_append_only ON wallet_transactions;
CREATE TRIGGER wt_append_only
  BEFORE UPDATE OR DELETE ON wallet_transactions
  FOR EACH ROW EXECUTE FUNCTION wt_no_mutate();

-- 4. Ledger outbox: entries pending external notification (R8 real-time forwarding).
CREATE TABLE IF NOT EXISTS ledger_outbox (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    seq         BIGINT NOT NULL UNIQUE,
    entry_hash  BYTEA  NOT NULL,
    sent_at     TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_ledger_outbox_unsent
  ON ledger_outbox(created_at) WHERE sent_at IS NULL;
