-- 010_auto_trading.up.sql
-- 自动化交易功能相关表

-- 策略调度任务表
CREATE TABLE strategy_schedules (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    strategy_id UUID NOT NULL REFERENCES strategies(id) ON DELETE CASCADE,
    account_id UUID NOT NULL REFERENCES mt_accounts(id) ON DELETE CASCADE,
    schedule_type VARCHAR(50) NOT NULL, -- 'cron', 'interval', 'event'
    schedule_config JSONB NOT NULL, -- cron表达式或间隔配置
    is_active BOOLEAN DEFAULT false,
    last_run_at TIMESTAMP,
    next_run_at TIMESTAMP,
    last_error TEXT,
    run_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 策略执行历史表
CREATE TABLE strategy_executions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    strategy_id UUID NOT NULL REFERENCES strategies(id) ON DELETE CASCADE,
    schedule_id UUID REFERENCES strategy_schedules(id) ON DELETE SET NULL,
    account_id UUID NOT NULL REFERENCES mt_accounts(id) ON DELETE CASCADE,
    status VARCHAR(50) NOT NULL, -- 'running', 'completed', 'failed', 'cancelled'
    signals JSONB, -- 生成的信号
    orders JSONB, -- 执行的订单
    error_message TEXT,
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP
);

-- 风险配置表
CREATE TABLE risk_configs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    account_id UUID REFERENCES mt_accounts(id) ON DELETE CASCADE, -- NULL表示全局配置
    max_risk_percent DECIMAL(5, 2) DEFAULT 2.00, -- 单笔最大风险百分比
    max_daily_loss DECIMAL(18, 2), -- 每日最大亏损金额
    max_drawdown_percent DECIMAL(5, 2), -- 最大回撤百分比
    max_positions INTEGER DEFAULT 5, -- 最大持仓数量
    max_lot_size DECIMAL(18, 6), -- 单笔最大手数
    daily_loss_used DECIMAL(18, 2) DEFAULT 0, -- 当日已用亏损额度
    trailing_stop_enabled BOOLEAN DEFAULT false,
    trailing_stop_pips DECIMAL(10, 2),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uk_user_account_risk UNIQUE (user_id, account_id)
);

-- 全局设置表
CREATE TABLE global_settings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE UNIQUE,
    auto_trade_enabled BOOLEAN DEFAULT false, -- 全局自动交易开关
    notification_enabled BOOLEAN DEFAULT true,
    email_notification BOOLEAN DEFAULT false,
    sms_notification BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 交易日志表（扩展）
CREATE TABLE trading_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    account_id UUID REFERENCES mt_accounts(id) ON DELETE SET NULL,
    strategy_id UUID REFERENCES strategies(id) ON DELETE SET NULL,
    execution_id UUID REFERENCES strategy_executions(id) ON DELETE SET NULL,
    log_type VARCHAR(50) NOT NULL, -- 'trade', 'signal', 'error', 'system'
    action VARCHAR(100) NOT NULL,
    symbol VARCHAR(20),
    details JSONB,
    message TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 索引
CREATE INDEX idx_strategy_schedules_strategy ON strategy_schedules(strategy_id);
CREATE INDEX idx_strategy_schedules_account ON strategy_schedules(account_id);
CREATE INDEX idx_strategy_schedules_active ON strategy_schedules(is_active);
CREATE INDEX idx_strategy_schedules_next_run ON strategy_schedules(next_run_at);

CREATE INDEX idx_strategy_executions_strategy ON strategy_executions(strategy_id);
CREATE INDEX idx_strategy_executions_account ON strategy_executions(account_id);
CREATE INDEX idx_strategy_executions_status ON strategy_executions(status);
CREATE INDEX idx_strategy_executions_started ON strategy_executions(started_at);

CREATE INDEX idx_risk_configs_user ON risk_configs(user_id);
CREATE INDEX idx_risk_configs_account ON risk_configs(account_id);

CREATE INDEX idx_global_settings_user ON global_settings(user_id);

CREATE INDEX idx_trading_logs_user ON trading_logs(user_id);
CREATE INDEX idx_trading_logs_account ON trading_logs(account_id);
CREATE INDEX idx_trading_logs_strategy ON trading_logs(strategy_id);
CREATE INDEX idx_trading_logs_type ON trading_logs(log_type);
CREATE INDEX idx_trading_logs_created ON trading_logs(created_at);

-- 触发器
CREATE TRIGGER update_strategy_schedules_updated_at BEFORE UPDATE ON strategy_schedules
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_risk_configs_updated_at BEFORE UPDATE ON risk_configs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_global_settings_updated_at BEFORE UPDATE ON global_settings
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- 注释
COMMENT ON TABLE strategy_schedules IS '策略调度任务表';
COMMENT ON TABLE strategy_executions IS '策略执行历史表';
COMMENT ON TABLE risk_configs IS '风险配置表';
COMMENT ON TABLE global_settings IS '用户全局设置表';
COMMENT ON TABLE trading_logs IS '交易日志表';

COMMENT ON COLUMN strategy_schedules.schedule_type IS '调度类型: cron-定时, interval-间隔, event-事件触发';
COMMENT ON COLUMN strategy_schedules.schedule_config IS '调度配置: cron表达式或间隔毫秒数';
COMMENT ON COLUMN strategy_executions.status IS '执行状态: running-运行中, completed-完成, failed-失败, cancelled-取消';
COMMENT ON COLUMN risk_configs.max_risk_percent IS '单笔交易最大风险百分比(基于账户余额)';
COMMENT ON COLUMN trading_logs.log_type IS '日志类型: trade-交易, signal-信号, error-错误, system-系统';
