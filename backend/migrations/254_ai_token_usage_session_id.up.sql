-- Phase 1.1: Add session_id to ai_token_usage for per-session attribution.
-- Nullable for backward compatibility with existing rows.
ALTER TABLE ai_token_usage ADD COLUMN IF NOT EXISTS session_id UUID;
CREATE INDEX IF NOT EXISTS idx_ai_token_usage_session_id ON ai_token_usage(session_id) WHERE session_id IS NOT NULL;
