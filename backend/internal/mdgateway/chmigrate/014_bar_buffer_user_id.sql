-- 014_bar_buffer_user_id.sql
-- M10.5: 011 added account_id to md_bars_buffer but missed user_id.
-- 012 added user_id to md_bars (ReplacingMergeTree) but not to its buffer.
-- When the buffer flushes, INSERT with user_id fails on the buffer table.
-- This adds the missing column to md_bars_buffer.

ALTER TABLE md_bars_buffer ADD COLUMN IF NOT EXISTS user_id LowCardinality(String);
