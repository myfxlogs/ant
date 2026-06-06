-- Add strategy_key column for strategy-scoped AI conversation sessions.
-- Uses partial unique index to enforce one session per strategy, nullable for
-- generic AI conversations (SystemAI page).

ALTER TABLE ai_conversations
  ADD COLUMN IF NOT EXISTS strategy_key VARCHAR(256);

CREATE UNIQUE INDEX IF NOT EXISTS idx_conv_strategy_key
  ON ai_conversations(user_id, strategy_key)
  WHERE strategy_key IS NOT NULL AND strategy_key != '';
