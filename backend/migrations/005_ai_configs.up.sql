-- 005_ai_configs.up.sql
-- AI配置表迁移

-- AI配置表
CREATE TABLE ai_configs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider VARCHAR(20) NOT NULL,
    api_key TEXT NOT NULL,
    model_name VARCHAR(100) NOT NULL,
    max_tokens INTEGER DEFAULT 4096,
    temperature DECIMAL(3, 2) DEFAULT 0.7,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uk_user_provider UNIQUE (user_id, provider)
);

CREATE INDEX idx_ai_configs_user ON ai_configs(user_id);
CREATE INDEX idx_ai_configs_provider ON ai_configs(provider);
CREATE INDEX idx_ai_configs_active ON ai_configs(is_active);

-- 添加更新时间触发器
CREATE TRIGGER update_ai_configs_updated_at BEFORE UPDATE ON ai_configs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- 添加注释
COMMENT ON TABLE ai_configs IS 'AI模型配置表';
COMMENT ON COLUMN ai_configs.id IS '配置ID';
COMMENT ON COLUMN ai_configs.user_id IS '用户ID';
COMMENT ON COLUMN ai_configs.provider IS 'AI提供商：zhipu, deepseek';
COMMENT ON COLUMN ai_configs.api_key IS 'API密钥（加密存储）';
COMMENT ON COLUMN ai_configs.model_name IS '模型名称';
COMMENT ON COLUMN ai_configs.max_tokens IS '最大Token数';
COMMENT ON COLUMN ai_configs.temperature IS '温度参数';
COMMENT ON COLUMN ai_configs.is_active IS '是否激活';
COMMENT ON COLUMN ai_configs.created_at IS '创建时间';
COMMENT ON COLUMN ai_configs.updated_at IS '更新时间';
