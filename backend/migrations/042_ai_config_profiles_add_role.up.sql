-- 为 AI 配置 Profile 增加 role 字段，支持 deep/quick/default 三种角色
-- deep   : 深度思考模型，用于策略生成、风险辩论等复杂分析任务
-- quick  : 快速响应模型，用于摘要生成、普通对话等简单任务
-- default: 通用（无特定角色，兼容旧配置）

ALTER TABLE ai_config_profiles
    ADD COLUMN IF NOT EXISTS role VARCHAR(20) NOT NULL DEFAULT 'default';

CREATE INDEX IF NOT EXISTS idx_ai_config_profiles_user_role
    ON ai_config_profiles(user_id, role);
