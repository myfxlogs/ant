-- 011_buffer_account_id.sql
-- M10.5: Add account_id column to buffer tables to match md_ticks/md_bars v2 schema.
-- The original 009/010 buffer migrations missed this column.

ALTER TABLE md_ticks_buffer ADD COLUMN IF NOT EXISTS account_id LowCardinality(String);
ALTER TABLE md_bars_buffer ADD COLUMN IF NOT EXISTS account_id LowCardinality(String);
