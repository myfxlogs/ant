-- 114_ai_conversations: AI strategy generation conversations (ADR-0017)

CREATE TABLE IF NOT EXISTS ai_conversations (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id       UUID NOT NULL,
    title         VARCHAR(200),
    state         VARCHAR(20) NOT NULL DEFAULT 'active', -- active, archived
    strategy_id   UUID,
    context_json  JSONB DEFAULT '{}',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    archived_at   TIMESTAMPTZ
);
CREATE INDEX idx_ai_conversations_user ON ai_conversations(user_id);

CREATE TABLE IF NOT EXISTS ai_messages (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    conversation_id UUID NOT NULL REFERENCES ai_conversations(id) ON DELETE CASCADE,
    role            VARCHAR(20) NOT NULL, -- user, assistant, system
    content         TEXT NOT NULL,
    metadata_json   JSONB DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_ai_messages_conv ON ai_messages(conversation_id);
