-- 117_tenant_id_and_config: add tenant_id to business tables + create tenant_config (M10-BASE-A4).

-- Add tenant_id to broker_symbols (existing data defaults to 'default').
ALTER TABLE broker_symbols ADD COLUMN IF NOT EXISTS tenant_id VARCHAR(64) NOT NULL DEFAULT 'default';

-- Add tenant_id to signal_configs (if the table exists).
DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'signal_configs') THEN
        ALTER TABLE signal_configs ADD COLUMN IF NOT EXISTS tenant_id VARCHAR(64) NOT NULL DEFAULT 'default';
    END IF;
END $$;

-- Add tenant_id to platform_strategies.
ALTER TABLE platform_strategies ADD COLUMN IF NOT EXISTS tenant_id VARCHAR(64) NOT NULL DEFAULT 'default';

-- Tenant configuration table: key/value store with versioning.
-- tenant_config dropped (166_cleanup_dead_tables)

-- Notify on config change so in-memory cache can hot-reload.
CREATE OR REPLACE FUNCTION notify_tenant_config_change() RETURNS TRIGGER AS $$
BEGIN
    PERFORM pg_notify('tenant_config_changed', json_build_object(
        'tenant_id', COALESCE(NEW.tenant_id, OLD.tenant_id),
        'config_key', COALESCE(NEW.config_key, OLD.config_key),
        'op', TG_OP
    )::text);
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_tenant_config_notify ON tenant_config;
CREATE TRIGGER trg_tenant_config_notify
    AFTER INSERT OR UPDATE OR DELETE ON tenant_config
    FOR EACH ROW EXECUTE FUNCTION notify_tenant_config_change();

-- Index for fast per-tenant lookup.
CREATE INDEX IF NOT EXISTS idx_tenant_config_tenant ON tenant_config (tenant_id);
