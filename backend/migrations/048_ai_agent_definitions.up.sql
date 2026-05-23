CREATE TABLE IF NOT EXISTS ai_agent_definitions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    profile_id UUID NOT NULL REFERENCES ai_config_profiles(id) ON DELETE CASCADE,
    agent_key TEXT NOT NULL,
    type TEXT NOT NULL,
    name TEXT NOT NULL,
    identity TEXT NOT NULL,
    input_hint TEXT NOT NULL DEFAULT '',
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    position INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT ck_ai_agent_definitions_type CHECK (type IN (
        'style',
        'signals',
        'risk',
        'macro',
        'sentiment',
        'portfolio',
        'execution',
        'code'
    )),
    CONSTRAINT uk_ai_agent_definitions_profile_key UNIQUE (profile_id, agent_key)
);

CREATE INDEX IF NOT EXISTS idx_ai_agent_definitions_user_profile ON ai_agent_definitions(user_id, profile_id);
CREATE INDEX IF NOT EXISTS idx_ai_agent_definitions_profile_position ON ai_agent_definitions(profile_id, position);
