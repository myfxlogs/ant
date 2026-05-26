-- 117 down: remove tenant columns and config table.

ALTER TABLE broker_symbols DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE signal_configs DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE platform_strategies DROP COLUMN IF EXISTS tenant_id;

DROP TRIGGER IF EXISTS trg_tenant_config_notify ON tenant_config;
DROP FUNCTION IF EXISTS notify_tenant_config_change();
DROP TABLE IF EXISTS tenant_config;
