-- 172_pg_trgm_search.up.sql
-- Enable pg_trgm extension for fuzzy marketplace search.

CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- GIN trigram indexes for fast similarity searches on marketplace listings.
CREATE INDEX IF NOT EXISTS idx_marketplace_title_trgm ON marketplace_strategies USING gin (title gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_marketplace_desc_trgm  ON marketplace_strategies USING gin (description gin_trgm_ops);

-- Trigram indexes on joined tables for publisher + strategy name search.
CREATE INDEX IF NOT EXISTS idx_templates_name_trgm   ON strategy_templates USING gin (name gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_users_nickname_trgm   ON users USING gin (nickname gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_users_email_trgm      ON users USING gin (email gin_trgm_ops);

COMMENT ON INDEX idx_marketplace_title_trgm IS 'Trigram index for fuzzy title search.';
COMMENT ON INDEX idx_marketplace_desc_trgm  IS 'Trigram index for fuzzy description search.';
COMMENT ON INDEX idx_templates_name_trgm   IS 'Trigram index for strategy name search.';
COMMENT ON INDEX idx_users_nickname_trgm   IS 'Trigram index for publisher nickname search.';
COMMENT ON INDEX idx_users_email_trgm      IS 'Trigram index for publisher email search.';
