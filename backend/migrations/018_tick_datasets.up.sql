-- 018_tick_datasets.up.sql
-- Tick datasets (quote ticks) for reproducible tick-driven backtests

CREATE TABLE IF NOT EXISTS tick_datasets (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL,
    account_id UUID NOT NULL,
    symbol VARCHAR(20) NOT NULL,
    from_time TIMESTAMP NOT NULL,
    to_time TIMESTAMP NOT NULL,
    frozen BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_tick_datasets_user ON tick_datasets(user_id);
CREATE INDEX IF NOT EXISTS idx_tick_datasets_account ON tick_datasets(account_id);
CREATE INDEX IF NOT EXISTS idx_tick_datasets_symbol ON tick_datasets(symbol);

CREATE TABLE IF NOT EXISTS tick_dataset_ticks (
    dataset_id UUID NOT NULL,
    time TIMESTAMP NOT NULL,
    bid DECIMAL(18, 8) NOT NULL,
    ask DECIMAL(18, 8) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_tick_dataset_ticks_dataset FOREIGN KEY (dataset_id) REFERENCES tick_datasets(id) ON DELETE CASCADE,
    CONSTRAINT uk_tick_dataset_ticks_unique UNIQUE (dataset_id, time)
);

CREATE INDEX IF NOT EXISTS idx_tick_dataset_ticks_dataset_time ON tick_dataset_ticks(dataset_id, time);
