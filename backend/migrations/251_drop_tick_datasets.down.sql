-- Recreate tick dataset tables (rollback of 251).
-- NOTE: Data will NOT be restored, only schema.
CREATE TABLE IF NOT EXISTS tick_datasets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    symbol TEXT NOT NULL,
    timeframe TEXT NOT NULL,
    broker TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS tick_dataset_ticks (
    id BIGSERIAL PRIMARY KEY,
    dataset_id UUID NOT NULL REFERENCES tick_datasets(id) ON DELETE CASCADE,
    ts TIMESTAMPTZ NOT NULL,
    bid NUMERIC(20,8),
    ask NUMERIC(20,8),
    volume NUMERIC(20,8)
);
