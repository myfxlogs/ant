-- 224_marketplace_featured.up.sql
ALTER TABLE marketplace_strategies ADD COLUMN IF NOT EXISTS is_featured BOOLEAN DEFAULT false;
ALTER TABLE marketplace_strategies ADD COLUMN IF NOT EXISTS featured_until TIMESTAMPTZ;
ALTER TABLE marketplace_strategies ADD COLUMN IF NOT EXISTS featured_priority INT DEFAULT 0;
