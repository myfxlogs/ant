-- 216_marketplace_disclaimer.up.sql
ALTER TABLE marketplace_strategies ADD COLUMN IF NOT EXISTS disclaimer TEXT;
