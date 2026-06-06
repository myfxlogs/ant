DROP INDEX IF EXISTS idx_conv_strategy_key;
ALTER TABLE ai_conversations DROP COLUMN IF EXISTS strategy_key;
