-- 021_ai_conversations.down.sql
-- Auto-generated rollback for 021_ai_conversations

-- Drop indexes
DROP INDEX IF EXISTS idx_ai_conversations_updated_at;
DROP INDEX IF EXISTS idx_ai_conversations_user_id;
DROP INDEX IF EXISTS idx_ai_messages_conversation_id;
DROP INDEX IF EXISTS idx_ai_messages_created_at;

-- Drop tables
DROP TABLE IF EXISTS ai_messages CASCADE;
DROP TABLE IF EXISTS ai_conversations CASCADE;

