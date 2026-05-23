-- 账户连接日志表
CREATE TABLE IF NOT EXISTS account_connection_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id),
    account_id UUID NOT NULL REFERENCES mt_accounts(id),
    event_type VARCHAR(50) NOT NULL,
    status VARCHAR(20) NOT NULL,
    message TEXT,
    error_detail TEXT,
    server_host VARCHAR(255),
    server_port INTEGER,
    login_id BIGINT,
    connection_duration_seconds BIGINT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_account_conn_logs_user ON account_connection_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_account_conn_logs_account ON account_connection_logs(account_id);
CREATE INDEX IF NOT EXISTS idx_account_conn_logs_event_type ON account_connection_logs(event_type);
CREATE INDEX IF NOT EXISTS idx_account_conn_logs_created_at ON account_connection_logs(created_at);

-- 策略执行日志表
CREATE TABLE IF NOT EXISTS strategy_execution_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id),
    schedule_id UUID,
    template_id UUID,
    account_id UUID REFERENCES mt_accounts(id),
    symbol VARCHAR(50) NOT NULL,
    timeframe VARCHAR(20) NOT NULL,
    status VARCHAR(20) NOT NULL,
    signal_type VARCHAR(20),
    signal_price DECIMAL(20, 8),
    signal_volume DECIMAL(20, 8),
    signal_stop_loss DECIMAL(20, 8),
    signal_take_profit DECIMAL(20, 8),
    executed_order_id VARCHAR(100),
    executed_price DECIMAL(20, 8),
    executed_volume DECIMAL(20, 8),
    profit DECIMAL(20, 8),
    error_message TEXT,
    execution_time_ms BIGINT,
    kline_data JSONB,
    strategy_params JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_strategy_exec_logs_user ON strategy_execution_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_strategy_exec_logs_schedule ON strategy_execution_logs(schedule_id);
CREATE INDEX IF NOT EXISTS idx_strategy_exec_logs_account ON strategy_execution_logs(account_id);
CREATE INDEX IF NOT EXISTS idx_strategy_exec_logs_status ON strategy_execution_logs(status);
CREATE INDEX IF NOT EXISTS idx_strategy_exec_logs_created_at ON strategy_execution_logs(created_at);

DO $$
BEGIN
    IF to_regclass('public.strategy_schedules_v2') IS NOT NULL THEN
        IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'strategy_execution_logs_schedule_id_fkey') THEN
            ALTER TABLE strategy_execution_logs
                ADD CONSTRAINT strategy_execution_logs_schedule_id_fkey
                FOREIGN KEY (schedule_id) REFERENCES strategy_schedules_v2(id);
        END IF;
    END IF;
    IF to_regclass('public.strategy_templates') IS NOT NULL THEN
        IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'strategy_execution_logs_template_id_fkey') THEN
            ALTER TABLE strategy_execution_logs
                ADD CONSTRAINT strategy_execution_logs_template_id_fkey
                FOREIGN KEY (template_id) REFERENCES strategy_templates(id);
        END IF;
    END IF;
END $$;

-- 订单历史表
CREATE TABLE IF NOT EXISTS order_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id),
    account_id UUID NOT NULL REFERENCES mt_accounts(id),
    ticket BIGINT NOT NULL,
    order_type VARCHAR(20) NOT NULL,
    symbol VARCHAR(50) NOT NULL,
    volume DECIMAL(20, 8) NOT NULL,
    open_price DECIMAL(20, 8),
    close_price DECIMAL(20, 8),
    open_time TIMESTAMP WITH TIME ZONE,
    close_time TIMESTAMP WITH TIME ZONE,
    stop_loss DECIMAL(20, 8),
    take_profit DECIMAL(20, 8),
    profit DECIMAL(20, 8),
    commission DECIMAL(20, 8),
    swap DECIMAL(20, 8),
    comment TEXT,
    magic_number BIGINT,
    is_auto_trade BOOLEAN DEFAULT FALSE,
    schedule_id UUID,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_order_history_user ON order_history(user_id);
CREATE INDEX IF NOT EXISTS idx_order_history_account ON order_history(account_id);
CREATE INDEX IF NOT EXISTS idx_order_history_ticket ON order_history(ticket);
CREATE INDEX IF NOT EXISTS idx_order_history_symbol ON order_history(symbol);
CREATE INDEX IF NOT EXISTS idx_order_history_open_time ON order_history(open_time);
CREATE INDEX IF NOT EXISTS idx_order_history_close_time ON order_history(close_time);

DO $$
BEGIN
    IF to_regclass('public.strategy_schedules_v2') IS NOT NULL THEN
        IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'order_history_schedule_id_fkey') THEN
            ALTER TABLE order_history
                ADD CONSTRAINT order_history_schedule_id_fkey
                FOREIGN KEY (schedule_id) REFERENCES strategy_schedules_v2(id);
        END IF;
    END IF;
END $$;

-- 系统操作日志表
CREATE TABLE IF NOT EXISTS system_operation_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id),
    operation_type VARCHAR(50) NOT NULL,
    module VARCHAR(50) NOT NULL,
    resource_type VARCHAR(50),
    resource_id UUID,
    action VARCHAR(100) NOT NULL,
    old_value JSONB,
    new_value JSONB,
    ip_address VARCHAR(50),
    user_agent TEXT,
    status VARCHAR(20) NOT NULL,
    error_message TEXT,
    duration_ms BIGINT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_system_op_logs_user ON system_operation_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_system_op_logs_operation ON system_operation_logs(operation_type);
CREATE INDEX IF NOT EXISTS idx_system_op_logs_module ON system_operation_logs(module);
CREATE INDEX IF NOT EXISTS idx_system_op_logs_resource ON system_operation_logs(resource_type, resource_id);
CREATE INDEX IF NOT EXISTS idx_system_op_logs_created_at ON system_operation_logs(created_at);

-- 插入到迁移记录表
INSERT INTO schema_migrations (version) VALUES ('012_enhanced_logs') ON CONFLICT DO NOTHING;
