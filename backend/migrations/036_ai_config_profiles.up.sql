CREATE TABLE IF NOT EXISTS ai_config_profiles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    provider VARCHAR(50) NOT NULL,
    api_key TEXT NOT NULL,
    model_name VARCHAR(100) NOT NULL,
    base_url TEXT NOT NULL DEFAULT '',
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    is_current BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_ai_config_profiles_user ON ai_config_profiles(user_id);
CREATE INDEX IF NOT EXISTS idx_ai_config_profiles_user_current ON ai_config_profiles(user_id, is_current);

CREATE UNIQUE INDEX IF NOT EXISTS uk_ai_config_profiles_user_current ON ai_config_profiles(user_id) WHERE is_current = TRUE;

CREATE UNIQUE INDEX IF NOT EXISTS uk_ai_config_profiles_user_name ON ai_config_profiles(user_id, name);

INSERT INTO ai_config_profiles (user_id, name, provider, api_key, model_name, base_url, enabled, is_current, created_at, updated_at)
SELECT user_id,
       '默认' as name,
       provider,
       api_key,
       model_name,
       COALESCE(base_url, '') as base_url,
       COALESCE(is_active, TRUE) as enabled,
       TRUE as is_current,
       created_at,
       updated_at
FROM ai_configs
WHERE is_active = TRUE
ON CONFLICT (user_id, name) DO NOTHING;
