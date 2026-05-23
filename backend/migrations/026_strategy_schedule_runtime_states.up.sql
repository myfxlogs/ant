CREATE TABLE IF NOT EXISTS strategy_schedule_runtime_states (
    schedule_id UUID PRIMARY KEY,
    state JSONB NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_strategy_schedule_runtime_states_updated_at ON strategy_schedule_runtime_states(updated_at);

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_trigger
        WHERE tgname = 'update_strategy_schedule_runtime_states_updated_at'
    ) THEN
        EXECUTE 'CREATE TRIGGER update_strategy_schedule_runtime_states_updated_at BEFORE UPDATE ON strategy_schedule_runtime_states FOR EACH ROW EXECUTE FUNCTION update_updated_at_column()';
    END IF;
END $$;
