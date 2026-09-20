-- 281_trade_records_dedup.up.sql
-- TRADE-RECORDS-DUP-1: deduplicate trade_records, archive-then-delete.
--
-- Two duplicate/phantom classes (2026-09-19 production-census figures) are
-- snapshotted into trade_record_dedup_log (full row incl. hash-chain
-- material) and deleted in the same transaction:
--
--   D1 'cst_utc_8h' (879 rows): the +8h side of verified UTC/CST twin pairs —
--       same account+ticket, close_time AND open_time both exactly +8h apart,
--       volume/open_price/close_price/profit identical. The UTC side is kept
--       (fact superset; only order_type letter-case differs) and logged as
--       kept_id. Exact-twin signature verified against the production copy:
--       879 pairs, zero field divergence beyond the documented ones.
--   D2 'epoch_zero' (3,024 rows): close_time='1970-01-01' — open-position
--       rows verbatim-copied from broker history into the closed-trade
--       ledger by the two unguarded sync sites (write-site guards land in
--       this change set, so deletion is not immediately undone by re-sync).
--
-- SET LOCAL session_replication_role='replica': DELETE on trade_records is
-- unconditionally rejected by prevent_trade_delete (migration 265); the
-- bypass is scoped to this migration's transaction only (precedent: 278).
-- Every deleted row is fully restorable from the log table — see the down
-- migration (seq is GENERATED ALWAYS AS IDENTITY and is re-inserted with
-- OVERRIDING SYSTEM VALUE at its original value, so the hash chain
-- reconstructs exactly).
--
-- Idempotent: after this migration runs, the twin join matches nothing and
-- the epoch rows are gone — re-running both DELETE CTEs removes 0 rows.

SET LOCAL session_replication_role = 'replica';

CREATE TABLE trade_record_dedup_log (
    log_id     BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    removed_id UUID NOT NULL,
    kept_id    UUID,             -- D1: id of the kept UTC twin; D2: NULL
    dup_class  TEXT NOT NULL,    -- 'cst_utc_8h' | 'epoch_zero'
    removed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    removed_by TEXT NOT NULL DEFAULT 'migration_281',
    -- Full-row snapshot of the removed trade_records row. Column order and
    -- types mirror trade_records' physical layout exactly: the INSERT below
    -- feeds `SELECT t.*` positionally, and the down migration re-inserts
    -- from these columns back into trade_records.
    id UUID, account_id UUID, ticket BIGINT,
    symbol VARCHAR(20), order_type VARCHAR(30),
    volume NUMERIC(10,4), open_price NUMERIC(18,8), close_price NUMERIC(18,8),
    profit NUMERIC(18,4), swap NUMERIC(18,4), commission NUMERIC(18,4),
    open_time TIMESTAMP, close_time TIMESTAMP,
    stop_loss NUMERIC(18,8), take_profit NUMERIC(18,8),
    order_comment VARCHAR(200), magic_number BIGINT, platform VARCHAR(10),
    created_at TIMESTAMP, updated_at TIMESTAMP,
    schedule_id UUID, user_id UUID,
    seq BIGINT, prev_hash BYTEA, entry_hash BYTEA
);

-- D1: archive then delete the +8h (CST-encoded) side of each twin pair.
-- The signature is the production-verified strict one: BOTH timestamps must
-- be exact +8h and all monetary fields identical (uk_trade_record_ticket on
-- account_id+ticket+close_time makes the doomed side unique per pair).
WITH doomed AS (
    SELECT b.id, a.id AS kept_id
    FROM trade_records a JOIN trade_records b
      ON b.account_id = a.account_id AND b.ticket = a.ticket
     AND b.close_time = a.close_time + interval '8 hours'
     AND b.open_time  = a.open_time  + interval '8 hours'
     AND b.volume = a.volume AND b.open_price = a.open_price
     AND b.close_price = a.close_price AND b.profit = a.profit
), ins AS (
    INSERT INTO trade_record_dedup_log (removed_id, kept_id, dup_class,
        id, account_id, ticket, symbol, order_type, volume, open_price, close_price,
        profit, swap, commission, open_time, close_time, stop_loss, take_profit,
        order_comment, magic_number, platform, created_at, updated_at,
        schedule_id, user_id, seq, prev_hash, entry_hash)
    SELECT d.id, d.kept_id, 'cst_utc_8h', t.* FROM doomed d JOIN trade_records t ON t.id = d.id
    RETURNING removed_id
)
DELETE FROM trade_records t USING ins WHERE t.id = ins.removed_id;

-- D2: archive then delete the epoch-zero phantom rows. Runs after D1 so a
-- row matching both classes is logged exactly once (as its first class).
WITH doomed AS (
    SELECT id FROM trade_records WHERE close_time = TIMESTAMP '1970-01-01'
), ins AS (
    INSERT INTO trade_record_dedup_log (removed_id, kept_id, dup_class,
        id, account_id, ticket, symbol, order_type, volume, open_price, close_price,
        profit, swap, commission, open_time, close_time, stop_loss, take_profit,
        order_comment, magic_number, platform, created_at, updated_at,
        schedule_id, user_id, seq, prev_hash, entry_hash)
    SELECT d.id, NULL, 'epoch_zero', t.* FROM doomed d JOIN trade_records t ON t.id = d.id
    RETURNING removed_id
)
DELETE FROM trade_records t USING ins WHERE t.id = ins.removed_id;
