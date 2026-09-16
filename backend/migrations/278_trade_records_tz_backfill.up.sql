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

UPDATE trade_records
SET close_time = close_time - interval '8 hours',
    open_time  = open_time  - interval '8 hours'
WHERE close_time - created_at BETWEEN interval '7 hours' AND interval '9 hours';
