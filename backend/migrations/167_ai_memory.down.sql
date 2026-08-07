-- 167_ai_memory.down.sql
-- Auto-generated rollback for 167_ai_memory

-- Drop indexes
DROP INDEX IF EXISTS idx_ai_memory_user;

-- Drop tables
DROP TABLE IF EXISTS ai_memory CASCADE;

