-- 005_ai_configs.down.sql
-- Auto-generated rollback for 005_ai_configs

-- Drop triggers
DROP TRIGGER IF EXISTS update_ai_configs_updated_at ON ai_configs;

-- Drop indexes
DROP INDEX IF EXISTS idx_ai_configs_active;
DROP INDEX IF EXISTS idx_ai_configs_provider;
DROP INDEX IF EXISTS idx_ai_configs_user;

-- Drop tables
DROP TABLE IF EXISTS ai_configs CASCADE;

