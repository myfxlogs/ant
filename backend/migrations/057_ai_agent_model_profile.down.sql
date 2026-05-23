DROP INDEX IF EXISTS idx_ai_agent_definitions_model_profile;
ALTER TABLE ai_agent_definitions DROP COLUMN IF EXISTS model_profile_id;
