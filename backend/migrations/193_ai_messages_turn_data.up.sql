-- 193: Add turn_data column to ai_messages for complete turn snapshot replay.
-- Stores proto-serialized AgentGenerateStrategyChunk (the final "done" chunk)
-- so frontend can reconstruct the full ChatTurn with code, metrics, and Apply button
-- when resuming a conversation from history.
ALTER TABLE ai_messages ADD COLUMN IF NOT EXISTS turn_data BYTEA;
