-- Down 060: 还原 ai_agent_definitions 与 profile 的耦合（有损）。
-- ai_config_profiles / ai_configs 仅恢复表结构，原数据丢失。
BEGIN;

-- 1) 重建被 drop 的两张表（仅最小骨架，足以让旧代码 compile）
CREATE TABLE IF NOT EXISTS ai_config_profiles (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    provider        TEXT NOT NULL,
    api_key         TEXT NOT NULL DEFAULT '',
    model_name      TEXT NOT NULL DEFAULT '',
    base_url        TEXT NOT NULL DEFAULT '',
    enabled         BOOLEAN NOT NULL DEFAULT TRUE,
    is_current      BOOLEAN NOT NULL DEFAULT FALSE,
    role            TEXT NOT NULL DEFAULT 'default',
    temperature     DOUBLE PRECISION,
    timeout_seconds INTEGER,
    max_tokens      INTEGER,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE IF NOT EXISTS ai_configs (
    user_id   UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    provider  TEXT NOT NULL,
    api_key   TEXT NOT NULL DEFAULT '',
    model     TEXT NOT NULL DEFAULT '',
    base_url  TEXT NOT NULL DEFAULT '',
    enabled   BOOLEAN NOT NULL DEFAULT TRUE
);

-- 2) ai_agent_definitions：恢复 profile_id + model_profile_id（数据无法回填）
ALTER TABLE ai_agent_definitions ADD COLUMN IF NOT EXISTS profile_id UUID;
ALTER TABLE ai_agent_definitions ADD COLUMN IF NOT EXISTS model_profile_id TEXT;

DROP INDEX IF EXISTS uk_ai_agent_definitions_user_key;
DROP INDEX IF EXISTS idx_ai_agent_definitions_user_position;

ALTER TABLE ai_agent_definitions DROP COLUMN IF EXISTS provider_id;
ALTER TABLE ai_agent_definitions DROP COLUMN IF EXISTS model_override;

COMMIT;
