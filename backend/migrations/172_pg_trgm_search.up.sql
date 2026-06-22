-- 172_pg_trgm_search.up.sql
-- Enable pg_trgm extension for fuzzy marketplace search.

CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- GIN trigram indexes for fast similarity searches on marketplace listings.
CREATE INDEX IF NOT EXISTS idx_marketplace_title_trgm ON marketplace_strategies USING gin (title gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_marketplace_desc_trgm  ON marketplace_strategies USING gin (description gin_trgm_ops);

COMMENT ON INDEX idx_marketplace_title_trgm IS 'Trigram index for fuzzy title search.';
COMMENT ON INDEX idx_marketplace_desc_trgm  IS 'Trigram index for fuzzy description search.';
