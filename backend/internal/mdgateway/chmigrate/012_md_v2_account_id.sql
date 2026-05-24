-- 012_md_v2_account_id.sql
-- M10.5-11: 011 only added account_id to md_ticks_buffer / md_bars_buffer; the
-- underlying ReplacingMergeTree tables md_ticks / md_bars were never updated,
-- so when the Buffer engine flushed it would fail with "No such column account_id".
-- This migration backfills the column on the v2 destination tables.
--
-- Discovered while implementing the M10.5-11 e2e smoke test: pipeline writes
-- failed with "code: 16, No such column account_id in table ant.md_ticks".

ALTER TABLE md_ticks ADD COLUMN IF NOT EXISTS account_id LowCardinality(String) AFTER user_id;
-- md_bars v1 schema lacked user_id; backfill it before account_id so v2 INSERT works.
ALTER TABLE md_bars  ADD COLUMN IF NOT EXISTS user_id    LowCardinality(String);
ALTER TABLE md_bars  ADD COLUMN IF NOT EXISTS account_id LowCardinality(String);
