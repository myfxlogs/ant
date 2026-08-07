-- 022_system_config_enabled.down.sql
ALTER TABLE system_config DROP COLUMN IF EXISTS enabled;
