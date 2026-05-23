-- Roll back: convert the TEXT locator back to UUID, dropping any non-profile
-- locators (e.g. `system:*`) since they cannot fit in a UUID column.
DROP INDEX IF EXISTS idx_ai_agent_definitions_model_profile;

UPDATE ai_agent_definitions
   SET model_profile_id = NULL
 WHERE model_profile_id IS NOT NULL
   AND model_profile_id NOT LIKE 'profile:%';

ALTER TABLE ai_agent_definitions
    ALTER COLUMN model_profile_id TYPE UUID
    USING (
        CASE
            WHEN model_profile_id IS NULL THEN NULL
            ELSE substring(model_profile_id from char_length('profile:') + 1)::uuid
        END
    );

ALTER TABLE ai_agent_definitions
    ADD CONSTRAINT ai_agent_definitions_model_profile_id_fkey
        FOREIGN KEY (model_profile_id) REFERENCES ai_config_profiles(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_ai_agent_definitions_model_profile
    ON ai_agent_definitions(model_profile_id);
