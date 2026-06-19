-- 164_gateway_default_model.up.sql
-- Add default_model column to system_ai_providers so admins can configure
-- a recommended model for each Gateway provider. If set, the resolution
-- logic prefers it over Models[0] when the user has not explicitly picked
-- a model via the Gateway picker (SetAIPrimary).

ALTER TABLE system_ai_providers
    ADD COLUMN IF NOT EXISTS default_model VARCHAR(200) NOT NULL DEFAULT '';
