-- Debate sessions: persist multi-agent debate conversations for the redesigned
-- conversational Debate module (see docs/debate_redesign.md).

CREATE TABLE IF NOT EXISTS debate_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(200) NOT NULL DEFAULT 'Debate',
    status VARCHAR(32) NOT NULL DEFAULT 'idle'
        CHECK (status IN (
            'idle',
            'clarifying',
            'intent_confirm',
            'debating',
            'consensus',
            'code_proposal',
            'saved',
            'archived'
        )),
    agents TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
    current_intent_turn_id UUID,
    current_consensus_turn_id UUID,
    current_code_turn_id UUID,
    template_id VARCHAR(64),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_debate_sessions_user_id ON debate_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_debate_sessions_updated_at ON debate_sessions(updated_at DESC);

CREATE TABLE IF NOT EXISTS debate_turns (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    session_id UUID NOT NULL REFERENCES debate_sessions(id) ON DELETE CASCADE,
    parent_turn_id UUID REFERENCES debate_turns(id) ON DELETE SET NULL,
    type VARCHAR(32) NOT NULL
        CHECK (type IN (
            'user_intent',
            'clarify_question',
            'clarify_answer',
            'intent_spec',
            'agent_opinion',
            'user_feedback',
            'consensus',
            'code_proposal',
            'system_note'
        )),
    role VARCHAR(32) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'approved'
        CHECK (status IN (
            'pending',
            'awaiting_user',
            'approved',
            'rejected',
            'superseded'
        )),
    content_text TEXT NOT NULL DEFAULT '',
    content_json JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_debate_turns_session_id ON debate_turns(session_id);
CREATE INDEX IF NOT EXISTS idx_debate_turns_created_at ON debate_turns(created_at);
CREATE INDEX IF NOT EXISTS idx_debate_turns_session_created ON debate_turns(session_id, created_at);
