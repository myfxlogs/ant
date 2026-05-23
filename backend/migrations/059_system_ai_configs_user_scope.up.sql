-- 059: 把 system_ai_configs 从「全局共享」改为「按用户隔离」。
-- 历史背景：056 引入此表时一行/厂商，所有用户共享同一份 API Key 与默认模型，
-- 不符合多用户隔离要求。本迁移给表加 user_id，并把现有的「实际配置过」的行
-- 归属到对应用户（updated_by 即配置者的 user uuid 字符串）。
-- 未被配置过的初始 stub（updated_by IS NULL OR ''）直接删除：每个用户首次
-- 进 /ai/settings 时由后端自动 seed。

BEGIN;

-- 1) 先加可空列，保留现有行以便归属。
ALTER TABLE system_ai_configs
    ADD COLUMN IF NOT EXISTS user_id UUID;

-- 2) 把有真实配置（updated_by 是合法 uuid）的行归属到对应用户。
UPDATE system_ai_configs
   SET user_id = updated_by::uuid
 WHERE updated_by IS NOT NULL
   AND updated_by <> ''
   AND updated_by ~* '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$';

-- 3) 删除从未被任何用户配置过的「全局 stub」行（无主、空配置）。
DELETE FROM system_ai_configs WHERE user_id IS NULL;

-- 4) 锁定 user_id 为 NOT NULL，并加外键级联（用户删除时清理其配置）。
ALTER TABLE system_ai_configs
    ALTER COLUMN user_id SET NOT NULL;

ALTER TABLE system_ai_configs
    ADD CONSTRAINT system_ai_configs_user_id_fkey
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

-- 5) 主键由 (provider_id) 改为 (user_id, provider_id)。
ALTER TABLE system_ai_configs DROP CONSTRAINT IF EXISTS system_ai_configs_pkey;
ALTER TABLE system_ai_configs ADD PRIMARY KEY (user_id, provider_id);

-- 6) 用户维度查询索引。
CREATE INDEX IF NOT EXISTS idx_system_ai_user ON system_ai_configs(user_id);

COMMIT;
