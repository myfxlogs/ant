-- 046_system_config_econ_translation.up.sql
-- Seed system_config rows for economic calendar translation settings so
-- they can be managed from the admin SystemConfig UI.

INSERT INTO system_config (key, value, description, enabled, updated_at)
VALUES
    ('econ.translation.zhipu_api_key', '', '智谱翻译用 API Key（经济日历事件名称本地化）', TRUE, CURRENT_TIMESTAMP),
    ('econ.translation.zhipu_model', 'glm-4-flash', '智谱翻译模型名称（默认 glm-4-flash）', TRUE, CURRENT_TIMESTAMP)
ON CONFLICT (key) DO UPDATE
SET value = EXCLUDED.value,
    description = EXCLUDED.description,
    enabled = EXCLUDED.enabled,
    updated_at = CURRENT_TIMESTAMP;
