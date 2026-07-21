-- 224_marketplace_featured.down.sql
ALTER TABLE marketplace_strategies DROP COLUMN IF EXISTS featured_priority;
ALTER TABLE marketplace_strategies DROP COLUMN IF EXISTS featured_until;
ALTER TABLE marketplace_strategies DROP COLUMN IF EXISTS is_featured;
