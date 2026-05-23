-- Down: 尽力恢复每位用户的 gemini 空行（与 EnsureSeed 默认一致）；不恢复已删密钥。
BEGIN;

INSERT INTO system_ai_configs (
    user_id, provider_id, name, base_url, organization, models, default_model,
    temperature, timeout_seconds, max_tokens, purposes, primary_for, enabled, updated_by
)
SELECT u.id, 'gemini', 'Google Gemini', '', '', '{}', '', 0.2, 60, 4096, '{}', '{}', FALSE, u.id::text
FROM users u
WHERE NOT EXISTS (
    SELECT 1 FROM system_ai_configs c
    WHERE c.user_id = u.id AND c.provider_id = 'gemini'
);

COMMIT;
