-- 191 down: recreate strategy_execution_logs (schema as of 012).
CREATE TABLE IF NOT EXISTS strategy_execution_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    account_id UUID NOT NULL REFERENCES mt_accounts(id) ON DELETE CASCADE,
    schedule_id UUID,
    symbol VARCHAR(20) NOT NULL,
    timeframe VARCHAR(10),
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    signal_type VARCHAR(20),
    signal_price NUMERIC(18,8),
    signal_volume NUMERIC(18,8),
    signal_stop_loss NUMERIC(18,8),
    signal_take_profit NUMERIC(18,8),
    executed_order_id BIGINT,
    executed_price NUMERIC(18,8),
    executed_volume NUMERIC(18,8),
    profit NUMERIC(18,2),
    error_message TEXT,
    execution_time_ms BIGINT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
