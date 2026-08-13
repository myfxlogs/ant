-- 275_fix_trade_records_hash_trigger.down.sql
-- Rollback: restore original trigger (blocks all hash field changes including NULL→non-NULL).

DROP TRIGGER IF EXISTS protect_trade_hash ON trade_records;
DROP FUNCTION IF EXISTS protect_trade_record_hash();

CREATE OR REPLACE FUNCTION protect_trade_record_hash()
RETURNS TRIGGER AS $$
BEGIN
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
