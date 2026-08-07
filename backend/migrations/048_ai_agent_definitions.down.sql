-- 048_ai_agent_definitions.down.sql
-- Auto-generated rollback for 048_ai_agent_definitions

-- Drop indexes
DROP INDEX IF EXISTS idx_ai_agent_definitions_profile_position;
DROP INDEX IF EXISTS idx_ai_agent_definitions_user_profile;

-- Drop tables
DROP TABLE IF EXISTS ai_agent_definitions CASCADE;

