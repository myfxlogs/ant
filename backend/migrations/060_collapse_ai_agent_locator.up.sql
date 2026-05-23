-- 060: 收口「AI 模型配置」单表化。
--
-- 历史上同一份「用户 → 模型」事实散落在 3 张表：
--   ai_configs（最早的 legacy，已空），
--   ai_config_profiles（用户多套 AI 档），
--   system_ai_configs（每用户每厂商一行，059 起 user-scope）。
-- 还有 ai_agent_definitions 用一个字符串 locator 同时引用前两者，导致：
--   * 数据重复、UI 多入口冲突；
--   * locator 字符串易野指针、跨表引用难维护；
--   * 前端「auto-seed 默认值」与「读库」竞态导致写丢失。
--
-- 本迁移：
--   1) ai_agent_definitions：去掉 profile_id / model_profile_id，
--      改为直接保存 provider_id（指向 system_ai_configs）+ 可选 model_override；
--   2) 把已存在的 locator 字符串解析到新列；
--   3) 解除与 ai_config_profiles 的 FK，drop 该表；同时 drop 早就空的 ai_configs。
--
-- ⚠️ 说明：ai_config_profiles 中的明文 api_key 不会自动迁入
-- system_ai_configs 的加密 secret——secretbox 加密只能在应用层完成。
-- 删除前 api_key 仍存在 ai_config_profiles 中（迁移后该表整体丢弃，
-- 视作"用户重新进 /ai/settings 录一次 key"）。当前生产数据仅 2 个 profile
-- 受此影响，且都仅用于过去的旧 UI；新 UI 已经在 system_ai_configs 里独立配置。

BEGIN;

-- 1) 新列：替代 model_profile_id locator
ALTER TABLE ai_agent_definitions
    ADD COLUMN IF NOT EXISTS provider_id TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS model_override TEXT NOT NULL DEFAULT '';

-- 2a) "system:<provider>" / "system:<provider>/<model>" → 新列
UPDATE ai_agent_definitions
   SET provider_id = SPLIT_PART(SUBSTRING(model_profile_id FROM 8), '/', 1),
       model_override = CASE
           WHEN POSITION('/' IN SUBSTRING(model_profile_id FROM 8)) > 0
           THEN SUBSTRING(model_profile_id FROM 8 + POSITION('/' IN SUBSTRING(model_profile_id FROM 8)))
           ELSE ''
       END
 WHERE model_profile_id IS NOT NULL
   AND model_profile_id LIKE 'system:%';

-- 2b) "profile:<uuid>" → 用对应 profile 的 provider/model_name 填新列
UPDATE ai_agent_definitions a
   SET provider_id = p.provider,
       model_override = COALESCE(p.model_name, '')
  FROM ai_config_profiles p
 WHERE a.model_profile_id IS NOT NULL
   AND a.model_profile_id LIKE 'profile:%'
   AND p.id::text = SUBSTRING(a.model_profile_id FROM 9)
   AND p.user_id = a.user_id;

-- 3) 清掉旧索引/约束/列
DROP INDEX IF EXISTS idx_ai_agent_definitions_model_profile;
DROP INDEX IF EXISTS idx_ai_agent_definitions_profile_position;
DROP INDEX IF EXISTS idx_ai_agent_definitions_user_profile;
ALTER TABLE ai_agent_definitions DROP CONSTRAINT IF EXISTS uk_ai_agent_definitions_profile_key;
ALTER TABLE ai_agent_definitions DROP CONSTRAINT IF EXISTS ai_agent_definitions_profile_id_fkey;
ALTER TABLE ai_agent_definitions DROP COLUMN IF EXISTS profile_id;
ALTER TABLE ai_agent_definitions DROP COLUMN IF EXISTS model_profile_id;

-- 4) 重新建立 user-scoped 唯一性 / 排序索引
CREATE UNIQUE INDEX IF NOT EXISTS uk_ai_agent_definitions_user_key
    ON ai_agent_definitions(user_id, agent_key);
CREATE INDEX IF NOT EXISTS idx_ai_agent_definitions_user_position
    ON ai_agent_definitions(user_id, "position");

-- 5) 丢掉两张废弃表
DROP TABLE IF EXISTS ai_config_profiles CASCADE;
DROP TABLE IF EXISTS ai_configs CASCADE;

-- 6) type CHECK 约束补 'custom'：UI 在「新增 Agent」时用此 type，
--    历史脚本一直没把它加进 ARRAY，新页面下能避免静默插入失败。
ALTER TABLE ai_agent_definitions DROP CONSTRAINT IF EXISTS ck_ai_agent_definitions_type;
ALTER TABLE ai_agent_definitions ADD CONSTRAINT ck_ai_agent_definitions_type
    CHECK (type = ANY (ARRAY['style','signals','risk','macro','sentiment','portfolio','execution','code','custom']));

COMMIT;
