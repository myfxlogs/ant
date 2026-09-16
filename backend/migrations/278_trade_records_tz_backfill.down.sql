-- 278_trade_records_tz_backfill.down.sql
-- TZ-MIXED-ENCODING-1 rollback — APPROXIMATE restore.
--
-- Re-applies the CST encoding to rows still matching the signature
-- (close_time - created_at in [7h,9h]). Rows already normalized by the
-- up migration no longer match the signature (their delta is ≈0), so
-- this cannot undo them — it only catches CST rows written between the
-- up run and a rollback (e.g. old binary still running in a deploy
-- window). For a precise restore use point-in-time recovery.

UPDATE trade_records
SET close_time = close_time + interval '8 hours',
    open_time  = open_time  + interval '8 hours'
WHERE close_time - created_at BETWEEN interval '7 hours' AND interval '9 hours';
