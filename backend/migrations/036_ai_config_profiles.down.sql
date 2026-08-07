-- 036_ai_config_profiles.down.sql
-- Auto-generated rollback for 036_ai_config_profiles

-- Drop indexes
DROP INDEX IF EXISTS idx_ai_config_profiles_user;
DROP INDEX IF EXISTS idx_ai_config_profiles_user_current;
DROP INDEX IF EXISTS uk_ai_config_profiles_user_current;
DROP INDEX IF EXISTS uk_ai_config_profiles_user_name;

-- Drop tables
DROP TABLE IF EXISTS ai_config_profiles CASCADE;

