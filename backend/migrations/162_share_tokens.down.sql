-- 162_share_tokens.down.sql
-- Auto-generated rollback for 162_share_tokens

-- Drop indexes
DROP INDEX IF EXISTS idx_share_tokens_token;
DROP INDEX IF EXISTS idx_share_tokens_user_id;

-- Drop tables
DROP TABLE IF EXISTS share_tokens CASCADE;

