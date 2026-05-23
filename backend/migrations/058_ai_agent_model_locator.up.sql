-- Switch the per-agent model override from a UUID FK into a free-form locator
-- string, so an agent can also point at a SystemAI provider (which is keyed by
-- string `provider_id`, not by ai_config_profiles.id).
--
-- Locator format:
--   profile:<uuid>     -> per-user ai_config_profiles row
--   system:<provider>  -> system_ai_configs row (provider_id)
--   ''/NULL            -> inherit role-based default
--
-- Existing rows are migrated to the `profile:<uuid>` form.

ALTER TABLE ai_agent_definitions
    DROP CONSTRAINT IF EXISTS ai_agent_definitions_model_profile_id_fkey;

DROP INDEX IF EXISTS idx_ai_agent_definitions_model_profile;

ALTER TABLE ai_agent_definitions
    ALTER COLUMN model_profile_id DROP DEFAULT;

ALTER TABLE ai_agent_definitions
    ALTER COLUMN model_profile_id TYPE TEXT
    USING (
        CASE
            WHEN model_profile_id IS NULL THEN NULL
            ELSE 'profile:' || model_profile_id::text
        END
    );

CREATE INDEX IF NOT EXISTS idx_ai_agent_definitions_model_profile
    ON ai_agent_definitions(model_profile_id);
