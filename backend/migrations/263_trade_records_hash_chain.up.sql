-- 263_trade_records_hash_chain.up.sql
-- FEAT-4: 实盘战绩不可篡改 — append-only hash chain for trade_records
--
-- Adds seq (global identity), prev_hash, entry_hash columns to trade_records.
-- The hash chain detects any post-insert modification to trade data,
-- supporting the "实盘战绩可信" product thesis.
--
-- Pattern follows wallet_transactions.ledgerChainInsert (wallet_repo.go:215).

ALTER TABLE trade_records
    ADD COLUMN seq BIGINT GENERATED ALWAYS AS IDENTITY,
    ADD COLUMN prev_hash BYTEA,
    ADD COLUMN entry_hash BYTEA;

-- Index for chain verification queries (per-account, ordered by seq).
CREATE INDEX idx_trade_records_seq ON trade_records(seq);
CREATE INDEX idx_trade_records_account_seq ON trade_records(account_id, seq);

-- Prevent modification of hash chain fields after insert.
-- Any UPDATE to prev_hash/entry_hash is blocked; seq is IDENTITY (immutable).
-- This ensures the chain can only be written at INSERT time.
CREATE OR REPLACE FUNCTION protect_trade_record_hash()
RETURNS TRIGGER AS $$
BEGIN
    -- Block any attempt to change hash fields on UPDATE.
    IF NEW.prev_hash IS DISTINCT FROM OLD.prev_hash
       OR NEW.entry_hash IS DISTINCT FROM OLD.entry_hash
       OR NEW.seq IS DISTINCT FROM OLD.seq THEN
        RAISE EXCEPTION 'trade_records hash chain fields are immutable (seq/prev_hash/entry_hash)';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER protect_trade_hash
    BEFORE UPDATE ON trade_records
    FOR EACH ROW
    EXECUTE FUNCTION protect_trade_record_hash();
