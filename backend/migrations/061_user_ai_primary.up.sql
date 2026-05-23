-- 061_user_ai_primary.up.sql
--
-- 引入「Default Primary Model」概念：
-- 用户在 /ai/settings 页面挑一条 system_ai_configs 记录作为「兜底大脑」。
--   * Clarify-Intent / 代码生成 等没绑特定 Agent 的步骤一律走它；
--   * Agent 没显式配 provider/model 时也回落到它；
--   * 模板编辑器里的「AI 助手 — 修改代码」也走它。
--
-- 用 (provider_id, model) 两列，而不是外键到 system_ai_configs.id：
--   - system_ai_configs 没单独主键（user_id + provider_id 联合标识），
--   - model 必须是该 provider 已启用的 models 之一，但不强制 FK，避免
--     用户改名 / 删模型时还要级联修复。
--
-- 不做数据回填：所有用户初始 ai_primary_provider_id = ''，调用方应解释为
-- 「未设置 → fallback 到 pickEnabledSystemAIRow 首行」，与 060 行为一致。

BEGIN;

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS ai_primary_provider_id text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS ai_primary_model       text NOT NULL DEFAULT '';

COMMIT;
