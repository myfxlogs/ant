-- 015_ai_gateway.down.sql
-- Auto-generated rollback for 015_ai_gateway

-- Drop indexes
DROP INDEX IF EXISTS idx_ai_token_usage_paid_by;
DROP INDEX IF EXISTS idx_ai_token_usage_user;

-- Drop tables
DROP TABLE IF EXISTS system_ai_providers CASCADE;
DROP TABLE IF EXISTS ai_token_usage CASCADE;
DROP TABLE IF EXISTS ai_models CASCADE;

