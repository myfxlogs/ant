-- 012_strategy_template_refactor.up.sql
-- 策略模板重构：将策略拆分为模板+调度任务两层

-- ============================================
-- 第一步：创建新表
-- ============================================

-- 策略模板表
CREATE TABLE IF NOT EXISTS strategy_templates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    code TEXT NOT NULL,
    parameters JSONB DEFAULT '{}',
    is_public BOOLEAN DEFAULT false,
    tags VARCHAR(100)[] DEFAULT '{}',
    use_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_timestamp,
    updated_at TIMESTAMP DEFAULT CURRENT_timestamp
);

-- 新调度任务表
CREATE TABLE IF NOT EXISTS strategy_schedules_v2 (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    template_id UUID NOT NULL REFERENCES strategy_templates(id) ON DELETE CASCADE,
    account_id UUID NOT NULL REFERENCES mt_accounts(id) ON DELETE CASCADE,
    
    name VARCHAR(255),
    symbol VARCHAR(20) NOT NULL,
    timeframe VARCHAR(10) NOT NULL DEFAULT 'H1',
    parameters JSONB DEFAULT '{}',
    
    schedule_type VARCHAR(20) NOT NULL DEFAULT 'interval',
    schedule_config JSONB NOT NULL DEFAULT '{}',
    
    backtest_metrics JSONB,
    risk_score INTEGER,
    risk_level VARCHAR(10),
    risk_reasons JSONB DEFAULT '[]',
    risk_warnings JSONB DEFAULT '[]',
    last_backtest_at TIMESTAMP,
    
    is_active BOOLEAN DEFAULT false,
    last_run_at TIMESTAMP,
    next_run_at TIMESTAMP,
    run_count INTEGER DEFAULT 0,
    last_error TEXT,
    
    created_at TIMESTAMP DEFAULT CURRENT_timestamp,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================
-- 第二步：创建索引
-- ============================================

CREATE INDEX IF NOT exists idx_strategy_templates_user_id ON strategy_templates(user_id);
CREATE INDEX IF not exists idx_strategy_templates_is_public ON strategy_templates(is_public);
CREATE INDEX IF not exists idx_strategy_templates_tags ON strategy_templates USING GIN(tags);
CREATE INDEX IF not exists idx_strategy_templates_name ON strategy_templates(name);

CREATE INDEX IF not exists idx_strategy_schedules_v2_user_id ON strategy_schedules_v2(user_id);
CREATE INDEX IF not exists idx_strategy_schedules_v2_template_id ON strategy_schedules_v2(template_id);
CREATE INDEX if not exists idx_strategy_schedules_v2_account_id ON strategy_schedules_v2(account_id);
CREATE INDEX IF not exists idx_strategy_schedules_v2_symbol ON strategy_schedules_v2(symbol);
CREATE INDEX if not exists idx_strategy_schedules_v2_is_active ON strategy_schedules_v2(is_active);
CREATE INDEX if not exists idx_strategy_schedules_v2_risk_level ON strategy_schedules_v2(risk_level);
CREATE INDEX if not exists idx_strategy_schedules_v2_next_run_at ON strategy_schedules_v2(next_run_at);

-- ============================================
-- 第三步：创建触发器
-- ============================================

CREATE TRIGGER IF not exists update_strategy_templates_updated_at BEFORE UPDATE ON strategy_templates
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER if not exists update_strategy_schedules_v2_updated_at BEFORE UPDATE ON strategy_schedules_v2
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================
-- 第四步：添加注释
-- ============================================

COMMENT ON TABLE strategy_templates IS '策略模板表 - 存储可复用的策略逻辑';
COMMENT ON TABLE strategy_schedules_v2 IS '策略调度任务表 - 策略模板的具体执行配置';

COMMENT ON COLUMN strategy_templates.parameters IS '参数定义 (JSON格式)';
COMMENT ON COLUMN strategy_templates.is_public IS '是否公开分享到模板市场';
COMMENT ON COLUMN strategy_templates.use_count IS '被使用次数';

COMMENT ON COLUMN strategy_schedules_v2.parameters IS '参数值 (用户配置)';
COMMENT ON COLUMN strategy_schedules_v2.backtest_metrics IS '回测指标快照';
COMMENT ON COLUMN strategy_schedules_v2.risk_score IS '风险评分 (0-100，越高风险越大)';
COMMENT ON COLUMN strategy_schedules_v2.risk_level IS '风险等级: low-低, medium-中, high-高, unknown-未知';
COMMENT ON COLUMN strategy_schedules_v2.risk_reasons IS '风险评估依据 (JSON数组)';
COMMENT ON COLUMN strategy_schedules_v2.risk_warnings IS '风险警告 (JSON数组)';
