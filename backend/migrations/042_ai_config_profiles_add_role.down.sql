-- 042_ai_config_profiles_add_role.down.sql
-- Auto-generated rollback for 042_ai_config_profiles_add_role

-- Drop indexes
DROP INDEX IF EXISTS idx_ai_config_profiles_user_role;

