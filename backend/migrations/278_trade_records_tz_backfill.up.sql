-- 278_trade_records_tz_backfill.up.sql
-- TZ-MIXED-ENCODING-1: normalize CST-encoded trade_records rows to UTC.
--
-- Background: live-stream writes used time.Unix(...) in Go Local (CST),
-- while history/import writes used proto AsTime() (UTC). pgx encodes
-- `timestamp` (non-timestamptz) params by wall-clock components, so the
-- live path stored values +8h ahead. Signature of a CST row:
-- close_time - created_at ≈ +8h (insert lag is seconds; a real close
-- cannot happen ~8h after the row's insert, so no false positives).
-- created_at itself is PG DEFAULT now() — always UTC.
--
-- Idempotent: after the UPDATE the delta is ≈0 and no longer matches
-- the signature, so re-running is safe. open_time is shifted on the same
-- rows (same write source; open_time has no reliable signature of its own).
--
-- session_replication_role='replica': the protect_trade_hash trigger
-- (migration 275) rejects ANY UPDATE on rows whose entry_hash IS NULL
-- (pre-chain rows), and this UPDATE must touch them. The trigger is only
-- bypassed for this transaction; hash fields are not modified, and
-- normalizing timestamps actually REPAIRS chain consistency — entry_hash
-- covers UnixMilli (location-independent epoch), so UTC-normalized
-- columns recompute to the same hash that was stored at write time.
SET LOCAL session_replication_role = 'replica';

-- NOT EXISTS guard: uk_trade_record_ticket (account_id, ticket, close_time)
-- — live-path CST rows that have a history-import UTC twin (same ticket and
-- true close instant) would collide on the unique key after the -8h shift.
-- Those are duplicate ledger rows of the same real trade; deduplicating an
-- append-only audit ledger (incl. 21 pairs where BOTH rows are already
-- hashed) is a separate design decision — TRADE-RECORDS-DUP-1 residual.
-- They are left unshifted (status quo) rather than deleted here.
UPDATE trade_records a
SET close_time = close_time - interval '8 hours',
    open_time  = open_time  - interval '8 hours'
WHERE close_time - created_at BETWEEN interval '7 hours' AND interval '9 hours'
  AND NOT EXISTS (
    SELECT 1 FROM trade_records b
    WHERE b.account_id = a.account_id
      AND b.ticket = a.ticket
      AND b.close_time = a.close_time - interval '8 hours'
      AND b.ctid <> a.ctid
  );
