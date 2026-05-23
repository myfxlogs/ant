-- 051_debate_sessions_param_schema.up.sql
-- Add a JSONB column to persist per-session strategy parameter schema for
-- Debate V2. The schema is stored as a JSON array of TemplateParameter
-- objects (see internal/model/strategy_template.go).

ALTER TABLE debate_sessions
    ADD COLUMN IF NOT EXISTS param_schema JSONB NOT NULL DEFAULT '[]'::JSONB;

COMMENT ON COLUMN debate_sessions.param_schema IS 'Strategy parameter schema (array of TemplateParameter JSON) for Debate V2 sessions.';
