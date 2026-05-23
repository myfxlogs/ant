-- 002_trade_logs.up.sql
-- 交易日志表

CREATE TABLE trade_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    account_id UUID NOT NULL REFERENCES mt_accounts(id) ON DELETE CASCADE,
    action VARCHAR(50) NOT NULL,
    symbol VARCHAR(20),
    order_type VARCHAR(20),
    volume DECIMAL(10, 4),
    price DECIMAL(18, 8),
    ticket BIGINT,
    profit DECIMAL(18, 4),
    message TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_trade_logs_user ON trade_logs(user_id);
CREATE INDEX idx_trade_logs_account ON trade_logs(account_id);
CREATE INDEX idx_trade_logs_action ON trade_logs(action);
CREATE INDEX idx_trade_logs_created_at ON trade_logs(created_at);
CREATE INDEX idx_trade_logs_symbol ON trade_logs(symbol);
