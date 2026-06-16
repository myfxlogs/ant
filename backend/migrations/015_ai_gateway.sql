-- AI Gateway: system-managed AI providers, models, and token usage tracking.
-- Migration: add system_ai_providers, ai_models, ai_token_usage tables.

CREATE TABLE IF NOT EXISTS system_ai_providers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    provider_id VARCHAR(50) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    base_url VARCHAR(500) NOT NULL,
    api_key_encrypted BYTEA,
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS ai_models (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    provider_id UUID NOT NULL REFERENCES system_ai_providers(id) ON DELETE CASCADE,
    model_name VARCHAR(100) NOT NULL,
    display_name VARCHAR(200),
    price_per_1m_input NUMERIC(10,8) NOT NULL DEFAULT 0,
    price_per_1m_output NUMERIC(10,8) NOT NULL DEFAULT 0,
    enabled BOOLEAN NOT NULL DEFAULT true,
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(provider_id, model_name)
);

CREATE TABLE IF NOT EXISTS ai_token_usage (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    wallet_transaction_id UUID REFERENCES wallet_transactions(id),
    paid_by VARCHAR(10) NOT NULL CHECK (paid_by IN ('system', 'user')),
    provider_id VARCHAR(50),
    model_name VARCHAR(100),
    feature VARCHAR(50) NOT NULL,
    input_tokens INT NOT NULL DEFAULT 0,
    output_tokens INT NOT NULL DEFAULT 0,
    cost NUMERIC(20,8),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ai_token_usage_user ON ai_token_usage(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ai_token_usage_paid_by ON ai_token_usage(paid_by, created_at DESC);

-- Seed default system providers (no API key yet — admin fills in).
INSERT INTO system_ai_providers (id, provider_id, name, base_url, enabled, created_at, updated_at)
VALUES
    (uuid_generate_v4(), 'deepseek', 'DeepSeek', 'https://api.deepseek.com/v1', true, NOW(), NOW()),
    (uuid_generate_v4(), 'openai', 'OpenAI', 'https://api.openai.com/v1', true, NOW(), NOW()),
    (uuid_generate_v4(), 'anthropic', 'Anthropic', 'https://api.anthropic.com/v1', true, NOW(), NOW()),
    (uuid_generate_v4(), 'qwen', '通义千问', 'https://dashscope.aliyuncs.com/compatible-mode/v1', true, NOW(), NOW()),
    (uuid_generate_v4(), 'moonshot', '月之暗面', 'https://api.moonshot.cn/v1', true, NOW(), NOW()),
    (uuid_generate_v4(), 'zhipu', '智谱 GLM', 'https://open.bigmodel.cn/api/paas/v4', true, NOW(), NOW())
ON CONFLICT (provider_id) DO NOTHING;
