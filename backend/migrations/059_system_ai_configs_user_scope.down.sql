-- Down: 把 system_ai_configs 还原为「按 provider 全局唯一」结构。
-- 多用户共存时只能保留最近一条；其它行被覆盖，因此 down 是有损的。
BEGIN;

DROP INDEX IF EXISTS idx_system_ai_user;
ALTER TABLE system_ai_configs DROP CONSTRAINT IF EXISTS system_ai_configs_pkey;

-- 同 provider 多行时仅保留 updated_at 最新的一条，避免新主键冲突。
DELETE FROM system_ai_configs a
USING system_ai_configs b
WHERE a.provider_id = b.provider_id
  AND a.user_id <> b.user_id
  AND a.updated_at < b.updated_at;

ALTER TABLE system_ai_configs DROP CONSTRAINT IF EXISTS system_ai_configs_user_id_fkey;
ALTER TABLE system_ai_configs DROP COLUMN IF EXISTS user_id;
ALTER TABLE system_ai_configs ADD PRIMARY KEY (provider_id);

COMMIT;
