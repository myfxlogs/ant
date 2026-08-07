-- 263_trade_records_hash_chain.down.sql

DROP TRIGGER IF EXISTS protect_trade_hash ON trade_records;
DROP FUNCTION IF EXISTS protect_trade_record_hash();

DROP INDEX IF EXISTS idx_trade_records_account_seq;
DROP INDEX IF EXISTS idx_trade_records_seq;

ALTER TABLE trade_records
    DROP COLUMN IF EXISTS entry_hash,
    DROP COLUMN IF EXISTS prev_hash,
    DROP COLUMN IF EXISTS seq;
