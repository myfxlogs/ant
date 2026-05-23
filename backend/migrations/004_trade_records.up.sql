-- 004_trade_records.up.sql
-- 交易记录表（历史订单）

CREATE TABLE trade_records (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    account_id UUID NOT NULL REFERENCES mt_accounts(id) ON DELETE CASCADE,
    ticket BIGINT NOT NULL,
    symbol VARCHAR(20) NOT NULL,
    order_type VARCHAR(10) NOT NULL,
    volume DECIMAL(10, 4) NOT NULL,
    open_price DECIMAL(18, 8) NOT NULL,
    close_price DECIMAL(18, 8) NOT NULL,
    profit DECIMAL(18, 4) DEFAULT 0,
    swap DECIMAL(18, 4) DEFAULT 0,
    commission DECIMAL(18, 4) DEFAULT 0,
    open_time TIMESTAMP NOT NULL,
    close_time TIMESTAMP NOT NULL,
    stop_loss DECIMAL(18, 8),
    take_profit DECIMAL(18, 8),
    order_comment VARCHAR(200),
    magic_number BIGINT DEFAULT 0,
    platform VARCHAR(10) NOT NULL DEFAULT 'MT4',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uk_trade_record_ticket UNIQUE (account_id, ticket, close_time)
);

CREATE INDEX idx_trade_records_account ON trade_records(account_id);
CREATE INDEX idx_trade_records_symbol ON trade_records(symbol);
CREATE INDEX idx_trade_records_close_time ON trade_records(close_time);
CREATE INDEX idx_trade_records_platform ON trade_records(platform);

CREATE TRIGGER update_trade_records_updated_at BEFORE UPDATE ON trade_records
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
