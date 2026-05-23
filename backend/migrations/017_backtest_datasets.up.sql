-- 017_backtest_datasets.up.sql
-- Backtest fixed datasets (snapshot) for reproducible backtests

CREATE TABLE IF NOT EXISTS backtest_datasets (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL,
    account_id UUID NOT NULL,
    symbol VARCHAR(20) NOT NULL,
    timeframe VARCHAR(10) NOT NULL,
    from_time TIMESTAMP,
    to_time TIMESTAMP,
    count INTEGER NOT NULL DEFAULT 0,
    frozen BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_backtest_datasets_user ON backtest_datasets(user_id);
CREATE INDEX IF NOT EXISTS idx_backtest_datasets_account ON backtest_datasets(account_id);
CREATE INDEX IF NOT EXISTS idx_backtest_datasets_symbol_tf ON backtest_datasets(symbol, timeframe);

CREATE TABLE IF NOT EXISTS backtest_dataset_bars (
    dataset_id UUID NOT NULL,
    symbol VARCHAR(20) NOT NULL,
    timeframe VARCHAR(10) NOT NULL,
    open_time TIMESTAMP NOT NULL,
    close_time TIMESTAMP NOT NULL,
    open_price DECIMAL(18, 8) NOT NULL,
    high_price DECIMAL(18, 8) NOT NULL,
    low_price DECIMAL(18, 8) NOT NULL,
    close_price DECIMAL(18, 8) NOT NULL,
    tick_volume BIGINT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_backtest_dataset_bars_dataset FOREIGN KEY (dataset_id) REFERENCES backtest_datasets(id) ON DELETE CASCADE,
    CONSTRAINT uk_backtest_dataset_bars_unique UNIQUE (dataset_id, open_time)
);

CREATE INDEX IF NOT EXISTS idx_backtest_dataset_bars_dataset_time ON backtest_dataset_bars(dataset_id, open_time);
