-- 035_ai_configs_add_base_url.down.sql
ALTER TABLE ai_configs DROP COLUMN IF EXISTS base_url;
