DROP INDEX IF EXISTS idx_ai_token_usage_session_id;
ALTER TABLE ai_token_usage DROP COLUMN IF EXISTS session_id;
