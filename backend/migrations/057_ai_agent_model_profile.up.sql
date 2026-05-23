-- Allow each agent to bind its own AI config profile (provider + model + key).
-- NULL means "fallback to the role-based provider (currently the user's default)".
ALTER TABLE ai_agent_definitions
    ADD COLUMN IF NOT EXISTS model_profile_id UUID
        REFERENCES ai_config_profiles(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_ai_agent_definitions_model_profile
    ON ai_agent_definitions(model_profile_id);
