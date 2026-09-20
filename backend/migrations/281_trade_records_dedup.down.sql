-- 281_trade_records_dedup.down.sql
-- TRADE-RECORDS-DUP-1 rollback: re-insert every archived row at its original
-- values — including the GENERATED ALWAYS seq (OVERRIDING SYSTEM VALUE) and
-- the original prev_hash/entry_hash — so the hash chain reconstructs exactly,
-- then drop the log table.
--
-- SET LOCAL session_replication_role='replica': same trigger bypass scope as
-- the up migration (design D7); no INSERT trigger exists on trade_records
-- today, the bypass keeps the rollback safe against future append-only
-- triggers.

SET LOCAL session_replication_role = 'replica';

INSERT INTO trade_records (
    id, account_id, ticket, symbol, order_type, volume,
    open_price, close_price, profit, swap, commission,
    open_time, close_time, stop_loss, take_profit,
    order_comment, magic_number, platform, created_at, updated_at,
    schedule_id, user_id, seq, prev_hash, entry_hash
)
OVERRIDING SYSTEM VALUE
SELECT id, account_id, ticket, symbol, order_type, volume,
       open_price, close_price, profit, swap, commission,
       open_time, close_time, stop_loss, take_profit,
       order_comment, magic_number, platform, created_at, updated_at,
       schedule_id, user_id, seq, prev_hash, entry_hash
FROM trade_record_dedup_log;

DROP TABLE trade_record_dedup_log;
