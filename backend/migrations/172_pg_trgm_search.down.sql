-- 172_pg_trgm_search.down.sql

DROP INDEX IF EXISTS idx_marketplace_title_trgm;
DROP INDEX IF EXISTS idx_marketplace_desc_trgm;
DROP INDEX IF EXISTS idx_marketplace_tags_trgm;
