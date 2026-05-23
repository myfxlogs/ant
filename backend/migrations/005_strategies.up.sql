-- 005_strategies.up.sql
-- AI交易策略助手相关表

-- 策略表
CREATE TABLE strategies (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    account_id UUID REFERENCES mt_accounts(id) ON DELETE SET NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    symbol VARCHAR(20) NOT NULL,
    conditions JSONB DEFAULT '[]',
    actions JSONB DEFAULT '[]',
    risk_control JSONB DEFAULT '{}',
    status VARCHAR(20) DEFAULT 'active',
    auto_execute BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 策略信号表
CREATE TABLE strategy_signals (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    strategy_id UUID NOT NULL REFERENCES strategies(id) ON DELETE CASCADE,
    account_id UUID NOT NULL,
    symbol VARCHAR(20) NOT NULL,
    signal_type VARCHAR(10) NOT NULL,  -- buy, sell, close
    volume DECIMAL(18, 6),
    price DECIMAL(18, 6),
    stop_loss DECIMAL(18, 6),
    take_profit DECIMAL(18, 6),
    reason TEXT,
    status VARCHAR(20) DEFAULT 'pending',  -- pending, confirmed, executed, cancelled
    executed_at TIMESTAMP,
    ticket BIGINT,
    profit DECIMAL(18, 6),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 索引
CREATE INDEX idx_strategies_user_id ON strategies(user_id);
CREATE INDEX idx_strategies_account_id ON strategies(account_id);
CREATE INDEX idx_strategies_symbol ON strategies(symbol);
CREATE INDEX idx_strategies_status ON strategies(status);
CREATE INDEX idx_strategy_signals_strategy_id ON strategy_signals(strategy_id);
CREATE INDEX idx_strategy_signals_account_id ON strategy_signals(account_id);
CREATE INDEX idx_strategy_signals_status ON strategy_signals(status);
CREATE INDEX idx_strategy_signals_created_at ON strategy_signals(created_at);

-- 触发器
CREATE TRIGGER update_strategies_updated_at BEFORE UPDATE ON strategies
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- 注释
COMMENT ON TABLE strategies IS 'AI交易策略表';
COMMENT ON TABLE strategy_signals IS '策略信号记录表';
COMMENT ON COLUMN strategies.conditions IS '策略条件数组(JSON格式)';
COMMENT ON COLUMN strategies.actions IS '交易动作数组(JSON格式)';
COMMENT ON COLUMN strategies.risk_control IS '风控参数(JSON格式)';
COMMENT ON COLUMN strategies.status IS '策略状态: active-激活, paused-暂停, stopped-停止';
COMMENT ON COLUMN strategies.auto_execute IS '是否自动执行交易';
COMMENT ON COLUMN strategy_signals.signal_type IS '信号类型: buy-买入, sell-卖出, close-平仓';
COMMENT ON COLUMN strategy_signals.status IS '信号状态: pending-待处理, confirmed-已确认, executed-已执行, cancelled-已取消';
