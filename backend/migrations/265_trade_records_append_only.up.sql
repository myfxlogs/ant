-- 265_trade_records_append_only.up.sql
-- FEAT-4V: Make trade_records truly append-only by blocking DELETE.
-- The hash chain trigger (263) already blocks UPDATE of hash fields.
-- This trigger blocks all DELETEs, ensuring records cannot be silently
-- removed from the tail of the chain.
--
-- Together with 263, this makes trade_records tamper-proof:
--   INSERT: allowed (hash chain extended)
--   UPDATE: blocked if hash fields (seq/prev_hash/entry_hash) change
--   DELETE: blocked entirely

CREATE OR REPLACE FUNCTION prevent_trade_record_delete()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'trade_records is append-only — DELETE is not permitted';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER prevent_trade_delete
    BEFORE DELETE ON trade_records
    FOR EACH ROW
    EXECUTE FUNCTION prevent_trade_record_delete();
