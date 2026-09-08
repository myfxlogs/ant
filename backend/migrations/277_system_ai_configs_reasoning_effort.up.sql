ALTER TABLE system_ai_configs
    ADD COLUMN IF NOT EXISTS reasoning_effort TEXT NOT NULL DEFAULT '';
