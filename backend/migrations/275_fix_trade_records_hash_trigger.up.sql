-- 275_fix_trade_records_hash_trigger.up.sql
-- EXEC-1: Fix trigger to allow entry_hash NULL→non-NULL (one-shot set after INSERT).
--
-- The original trigger (migration 263) used `IS DISTINCT FROM` which blocks
-- NULL→non-NULL changes, preventing entry_hash from ever being set after INSERT.
-- This mirrors the wallet_transactions trigger pattern (migration 209):
-- allow entry_hash to be set once from NULL→non-NULL, block all other hash field changes.

DROP TRIGGER IF EXISTS protect_trade_hash ON trade_records;
DROP FUNCTION IF EXISTS protect_trade_record_hash();

CREATE OR REPLACE FUNCTION protect_trade_record_hash()
RETURNS TRIGGER AS $$
BEGIN
    -- Allow one-shot entry_hash set: NULL → non-NULL.
    -- Once set (non-NULL), any change is blocked.
    IF OLD.entry_hash IS NOT NULL THEN
        IF NEW.entry_hash IS DISTINCT FROM OLD.entry_hash THEN
            RAISE EXCEPTION 'trade_records hash chain fields are immutable (entry_hash already set)';
        END IF;
    ELSE
        -- entry_hash was NULL: allow setting to non-NULL, block clearing.
        IF NEW.entry_hash IS NULL THEN
            RAISE EXCEPTION 'trade_records hash chain: cannot clear entry_hash';
        END IF;
    END IF;

    -- prev_hash and seq are always immutable.
    IF NEW.prev_hash IS DISTINCT FROM OLD.prev_hash
       OR NEW.seq IS DISTINCT FROM OLD.seq THEN
        RAISE EXCEPTION 'trade_records hash chain fields are immutable (seq/prev_hash)';
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER protect_trade_hash
    BEFORE UPDATE ON trade_records
    FOR EACH ROW
    EXECUTE FUNCTION protect_trade_record_hash();
