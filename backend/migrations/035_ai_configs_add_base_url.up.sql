-- 035_ai_configs_add_base_url.up.sql
-- Add base_url for OpenAI-compatible/custom providers

ALTER TABLE ai_configs
    ADD COLUMN IF NOT EXISTS base_url TEXT NOT NULL DEFAULT '';
