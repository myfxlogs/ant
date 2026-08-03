-- Phase 3.1: Agent trajectory data collection.
-- Tracks fine-grained agent execution events for analysis, replay, and quality improvement.
-- Each event = one step in the agent loop (reasoning, tool call, code generation, backtest, etc.)

CREATE TABLE IF NOT EXISTS agent_trajectory_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id      UUID NOT NULL,
    user_id         UUID NOT NULL,
    event_seq       INT NOT NULL,
    event_type      TEXT NOT NULL,
    -- event_type values: 'reasoning', 'tool_call', 'tool_result', 'code_generated',
    -- 'backtest_started', 'backtest_completed', 'compile_error', 'user_feedback',
    -- 'signal_dispatched', 'circuit_breaker_tripped'
    content         TEXT,
    -- JSONB metadata for structured event details (tool name, args, result, metrics, etc.)
    -- Managed by DB, not application-layer json.Marshal (per project rules: JSONB is OK).
    metadata        JSONB,
    token_input     INT DEFAULT 0,
    token_output    INT DEFAULT 0,
    cost            NUMERIC(12,8) DEFAULT 0,
    duration_ms     INT DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Foreign key to ai_conversations (session = conversation).
    FOREIGN KEY (session_id) REFERENCES ai_conversations(id) ON DELETE CASCADE,

    -- One event per (session, seq) — prevents duplicate insertion.
    UNIQUE (session_id, event_seq)
);

-- Index for querying events by session in order.
CREATE INDEX IF NOT EXISTS idx_agent_trajectory_session
    ON agent_trajectory_events(session_id, event_seq);

-- Index for filtering by event type (e.g. all compile errors).
CREATE INDEX IF NOT EXISTS idx_agent_trajectory_type
    ON agent_trajectory_events(event_type)
    WHERE event_type IN ('compile_error', 'circuit_breaker_tripped', 'backtest_completed');

-- Index for cost analysis per user.
CREATE INDEX IF NOT EXISTS idx_agent_trajectory_user_cost
    ON agent_trajectory_events(user_id, created_at)
    WHERE cost > 0;
