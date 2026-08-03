-- Phase 1.3a: Session-level user feedback for AI generation sessions.
-- Captures explicit good/bad rating + optional reason, linked to ai_conversations.
-- This is the flywheel's explicit annotation signal (remediation plan §1.3).

CREATE TABLE IF NOT EXISTS ai_session_feedback (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id      UUID NOT NULL,
    user_id         UUID NOT NULL,
    rating          TEXT NOT NULL,  -- 'good' or 'bad'
    reason          TEXT,           -- optional free-text reason
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- One feedback per session per user (upsert).
    UNIQUE (session_id, user_id),

    -- FK to ai_conversations.
    FOREIGN KEY (session_id) REFERENCES ai_conversations(id) ON DELETE CASCADE
);

-- Index for querying feedback by user.
CREATE INDEX IF NOT EXISTS idx_ai_session_feedback_user
    ON ai_session_feedback(user_id, created_at DESC);

-- Index for aggregating feedback stats.
CREATE INDEX IF NOT EXISTS idx_ai_session_feedback_rating
    ON ai_session_feedback(rating);
