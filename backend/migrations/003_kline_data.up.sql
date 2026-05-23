-- 003_kline_data.up.sql
-- K线数据表

CREATE TABLE IF NOT EXISTS kline_data (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    symbol VARCHAR(20) NOT NULL,
    timeframe VARCHAR(10) NOT NULL,
    open_time TIMESTAMP NOT NULL,
    close_time TIMESTAMP NOT NULL,
    open_price DECIMAL(18, 8) NOT NULL,
    high_price DECIMAL(18, 8) NOT NULL,
    low_price DECIMAL(18, 8) NOT NULL,
    close_price DECIMAL(18, 8) NOT NULL,
    tick_volume BIGINT DEFAULT 0,
    real_volume DECIMAL(18, 2) DEFAULT 0,
    spread INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uk_kline_symbol_tf_time UNIQUE (symbol, timeframe, open_time)
);

CREATE INDEX IF NOT EXISTS idx_kline_symbol ON kline_data(symbol);
CREATE INDEX IF NOT EXISTS idx_kline_timeframe ON kline_data(timeframe);
CREATE INDEX IF NOT EXISTS idx_kline_open_time ON kline_data(open_time);
CREATE INDEX IF NOT EXISTS idx_kline_symbol_tf_time ON kline_data(symbol, timeframe, open_time);

CREATE TABLE IF NOT EXISTS schema_migrations (
    version VARCHAR(255) PRIMARY KEY,
    applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO schema_migrations (version) VALUES 
('001_init'),
('002_trade_logs'),
('003_kline_data')
ON CONFLICT (version) DO NOTHING;
