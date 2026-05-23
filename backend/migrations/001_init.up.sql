-- 001_init.up.sql
-- 初始化数据库表结构

-- 启用UUID扩展
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 用户表
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    nickname VARCHAR(100),
    avatar VARCHAR(500),
    role VARCHAR(20) NOT NULL DEFAULT 'user',
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    last_login_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_status ON users(status);

-- MT账户表
CREATE TABLE mt_accounts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    mt_type VARCHAR(10) NOT NULL,
    broker_company VARCHAR(100),
    broker_server VARCHAR(100),
    broker_host VARCHAR(100) NOT NULL,
    login VARCHAR(50) NOT NULL,
    password VARCHAR(255) NOT NULL,
    alias VARCHAR(100),
    is_disabled BOOLEAN DEFAULT FALSE,
    balance DECIMAL(18, 2) DEFAULT 0,
    credit DECIMAL(18, 2) DEFAULT 0,
    equity DECIMAL(18, 2) DEFAULT 0,
    margin DECIMAL(18, 2) DEFAULT 0,
    free_margin DECIMAL(18, 2) DEFAULT 0,
    margin_level DECIMAL(10, 2) DEFAULT 0,
    leverage INTEGER DEFAULT 100,
    currency VARCHAR(10) DEFAULT 'USD',
    account_method VARCHAR(20),
    is_investor BOOLEAN DEFAULT FALSE,
    account_status VARCHAR(20) NOT NULL DEFAULT 'disconnected',
    stream_status VARCHAR(20) NOT NULL DEFAULT 'inactive',
    mt_token VARCHAR(500),
    last_error TEXT,
    last_connected_at TIMESTAMP,
    last_checked_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uk_user_mttype_login UNIQUE (user_id, mt_type, login)
);

CREATE INDEX idx_mt_accounts_user ON mt_accounts(user_id);
CREATE INDEX idx_mt_accounts_mt_type ON mt_accounts(mt_type);
CREATE INDEX idx_mt_accounts_status ON mt_accounts(account_status);

-- 持仓表
CREATE TABLE positions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    mt_account_id UUID NOT NULL REFERENCES mt_accounts(id) ON DELETE CASCADE,
    platform VARCHAR(10) NOT NULL DEFAULT 'MT4',
    ticket BIGINT NOT NULL,
    symbol VARCHAR(20) NOT NULL,
    order_type SMALLINT NOT NULL,
    volume DECIMAL(10, 2) NOT NULL,
    open_price DECIMAL(18, 8) NOT NULL,
    current_price DECIMAL(18, 8),
    stop_loss DECIMAL(18, 8),
    take_profit DECIMAL(18, 8),
    open_time TIMESTAMP NOT NULL,
    profit DECIMAL(18, 2) DEFAULT 0,
    swap DECIMAL(18, 2) DEFAULT 0,
    commission DECIMAL(18, 2) DEFAULT 0,
    fee DECIMAL(18, 2) DEFAULT 0,
    order_comment VARCHAR(100),
    magic_number INTEGER,
    close_reason VARCHAR(50),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uk_position_ticket UNIQUE (mt_account_id, ticket)
);

CREATE INDEX idx_positions_mt_account ON positions(mt_account_id);
CREATE INDEX idx_positions_symbol ON positions(symbol);
CREATE INDEX idx_positions_platform ON positions(platform);

-- 挂单表
CREATE TABLE orders (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    mt_account_id UUID NOT NULL REFERENCES mt_accounts(id) ON DELETE CASCADE,
    platform VARCHAR(10) NOT NULL DEFAULT 'MT4',
    ticket BIGINT NOT NULL,
    symbol VARCHAR(20) NOT NULL,
    order_type SMALLINT NOT NULL,
    volume DECIMAL(10, 2) NOT NULL,
    price DECIMAL(18, 8) NOT NULL,
    stop_limit_price DECIMAL(18, 8),
    stop_loss DECIMAL(18, 8),
    take_profit DECIMAL(18, 8),
    expiration TIMESTAMP,
    expiration_type VARCHAR(20) DEFAULT 'GTC',
    placed_type VARCHAR(20) DEFAULT 'Client',
    order_comment VARCHAR(100),
    magic_number INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uk_order_ticket UNIQUE (mt_account_id, ticket)
);

CREATE INDEX idx_orders_mt_account ON orders(mt_account_id);
CREATE INDEX idx_orders_symbol ON orders(symbol);
CREATE INDEX idx_orders_platform ON orders(platform);

-- 品种信息表
CREATE TABLE symbols (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    symbol VARCHAR(20) NOT NULL UNIQUE,
    description VARCHAR(200),
    digits INTEGER NOT NULL DEFAULT 5,
    contract_size DECIMAL(18, 2),
    min_lot DECIMAL(10, 2) DEFAULT 0.01,
    max_lot DECIMAL(10, 2) DEFAULT 100,
    lot_step DECIMAL(10, 2) DEFAULT 0.01,
    spread DECIMAL(10, 5),
    swap_long DECIMAL(10, 5),
    swap_short DECIMAL(10, 5),
    margin_initial DECIMAL(10, 2),
    margin_maintenance DECIMAL(10, 2),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_symbols_symbol ON symbols(symbol);

-- 实时报价缓存表
CREATE TABLE market_quotes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    symbol VARCHAR(20) NOT NULL UNIQUE,
    bid DECIMAL(18, 8),
    ask DECIMAL(18, 8),
    last DECIMAL(18, 8),
    volume DECIMAL(18, 2),
    high DECIMAL(18, 8),
    low DECIMAL(18, 8),
    server_time TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_quotes_symbol ON market_quotes(symbol);

-- 系统配置表
CREATE TABLE system_config (
    key VARCHAR(100) PRIMARY KEY,
    value TEXT NOT NULL,
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 插入默认配置
INSERT INTO system_config (key, value, description) VALUES
('max_accounts_per_user', '10', '每用户最大账户数'),
('max_positions_per_account', '50', '每账户最大持仓数'),
('default_leverage', '100', '默认杠杆'),
('min_lot_size', '0.01', '最小手数');

-- 创建更新时间触发器函数
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- 为需要的表添加触发器
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_mt_accounts_updated_at BEFORE UPDATE ON mt_accounts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_positions_updated_at BEFORE UPDATE ON positions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_orders_updated_at BEFORE UPDATE ON orders
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_symbols_updated_at BEFORE UPDATE ON symbols
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
