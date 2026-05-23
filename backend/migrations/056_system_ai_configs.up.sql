-- 056: system-level AI provider configurations
-- One row per provider type. Secrets stored encrypted (AES-GCM with per-row salt+nonce).
-- Coexists with per-user ai_config_profiles (no breaking change).
CREATE TABLE IF NOT EXISTS system_ai_configs (
    provider_id        TEXT PRIMARY KEY,
    name               TEXT NOT NULL,
    base_url           TEXT NOT NULL DEFAULT '',
    organization       TEXT NOT NULL DEFAULT '',
    models             TEXT[] NOT NULL DEFAULT '{}',
    default_model      TEXT NOT NULL DEFAULT '',
    temperature        DOUBLE PRECISION NOT NULL DEFAULT 0.2,
    timeout_seconds    INTEGER NOT NULL DEFAULT 60,
    max_tokens         INTEGER NOT NULL DEFAULT 4096,
    purposes           TEXT[] NOT NULL DEFAULT '{}',
    primary_for        TEXT[] NOT NULL DEFAULT '{}',

    secret_ciphertext  BYTEA,
    secret_salt        BYTEA,
    secret_nonce       BYTEA,
    has_secret         BOOLEAN NOT NULL DEFAULT FALSE,

    enabled            BOOLEAN NOT NULL DEFAULT FALSE,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by         TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_system_ai_enabled ON system_ai_configs(enabled);
CREATE INDEX IF NOT EXISTS idx_system_ai_primary ON system_ai_configs USING GIN(primary_for);

INSERT INTO system_ai_configs (provider_id, name, enabled, has_secret) VALUES
    ('openai',            'OpenAI',             FALSE, FALSE),
    ('anthropic',         'Anthropic (Claude)', FALSE, FALSE),
    ('deepseek',          'DeepSeek',           FALSE, FALSE),
    ('qwen',              '通义千问',            FALSE, FALSE),
    ('moonshot',          '月之暗面 (Kimi)',     FALSE, FALSE),
    ('zhipu',             '智谱 GLM',            FALSE, FALSE),
    ('openai_compatible', '自定义 (OpenAI 兼容)', FALSE, FALSE)
ON CONFLICT (provider_id) DO NOTHING;
