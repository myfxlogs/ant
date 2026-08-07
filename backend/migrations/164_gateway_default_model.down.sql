-- 164_gateway_default_model.down.sql
ALTER TABLE system_ai_providers DROP COLUMN IF EXISTS default_model;
