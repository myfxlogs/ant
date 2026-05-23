ALTER TABLE ai_config_profiles
    DROP COLUMN IF EXISTS temperature,
    DROP COLUMN IF EXISTS timeout_seconds,
    DROP COLUMN IF EXISTS max_tokens;
