-- Add user_id and template_id columns that the model expects but were missing from the schema.
ALTER TABLE strategy_executions ADD COLUMN IF NOT EXISTS user_id UUID REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE strategy_executions ADD COLUMN IF NOT EXISTS template_id UUID;

-- Add index for user-scoped queries (auto-trading status/executions)
CREATE INDEX IF NOT EXISTS idx_strategy_executions_user ON strategy_executions(user_id);
CREATE INDEX IF NOT EXISTS idx_strategy_executions_template ON strategy_executions(template_id);
