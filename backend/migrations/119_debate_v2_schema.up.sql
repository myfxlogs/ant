-- 119: Add Debate V2 columns to debate_sessions.
-- Stores steps (JSONB array), generated code, model info, and token usage.

ALTER TABLE debate_sessions
    ADD COLUMN IF NOT EXISTS steps JSONB NOT NULL DEFAULT '[]'::JSONB,
    ADD COLUMN IF NOT EXISTS code JSONB,
    ADD COLUMN IF NOT EXISTS provider TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS model TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS usage JSONB;

COMMENT ON COLUMN debate_sessions.steps IS 'Debate V2 steps: array of {stepKey, agentKey, agentName, messages[]}';
COMMENT ON COLUMN debate_sessions.code IS 'Generated strategy code: {text, python}';
COMMENT ON COLUMN debate_sessions.usage IS 'Cumulative LLM token usage: {prompt_tokens, completion_tokens, total_tokens}';
