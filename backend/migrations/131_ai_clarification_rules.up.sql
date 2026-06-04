-- 131_ai_clarification_rules: intent clarification rules for AI strategy generation (spec/26 Phase 1)

CREATE TABLE IF NOT EXISTS ai_clarification_rules (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    keywords    JSONB NOT NULL DEFAULT '[]',
    questions   JSONB NOT NULL DEFAULT '[]',
    param_map   JSONB NOT NULL DEFAULT '{}',
    priority    INTEGER NOT NULL DEFAULT 0,
    is_active   BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_ai_clarification_rules_active ON ai_clarification_rules(is_active, priority DESC);

-- Seed: 3 clarification rules

INSERT INTO ai_clarification_rules (keywords, questions, param_map, priority) VALUES

-- 1. 稳健/保守/低风险
('["稳健", "保守", "低风险", "安全", "稳当", "保险", "防御"]',
 '["您能接受的最大回撤是多少？（例如：10% 以内）",
   "您偏好什么持仓周期？（日内/短线/中线/长线）",
   "您更看重收益稳定还是高回报？"]'::jsonb,
 '{"max_drawdown": "0.10", "min_period": "1h", "risk_level": "low"}'::jsonb,
 10),

-- 2. 进攻/激进/高风险
('["进攻", "激进", "高风险", "高收益", "暴力", "猛", "刺激", "快速"]',
 '["您能接受的最大回撤是多少？（例如：30%）",
   "是否允许日内高频交易？",
   "如果连续亏损，您能承受多大的浮亏？"]'::jsonb,
 '{"max_drawdown": "0.30", "max_period": "15m", "risk_level": "high"}'::jsonb,
 10),

-- 3. 波段/震荡/高抛低吸
('["波段", "高抛低吸", "震荡", "做T", "短线", "日内", "快进快出"]',
 '["您想操作的品种是什么？（BTC/ETH/外汇/指数）",
   "预计持仓时间是几小时还是几天？",
   "您更偏好单品种还是多品种操作？"]'::jsonb,
 '{"strategy_family": "mean_reversion", "max_period": "4h"}'::jsonb,
 10);
