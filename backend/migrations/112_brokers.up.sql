-- 112_brokers.up.sql
-- M10: Admin-configurable mtapi gateway endpoints per broker (alfq pattern).
-- An empty mtapi_endpoint means "use the mtapi.io public gateway".
-- When broker_host on mt_accounts was previously pointing to a custom IP,
-- those values are migrated into brokers.mtapi_endpoint.

CREATE TABLE IF NOT EXISTS brokers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    code VARCHAR(50) NOT NULL,
    name VARCHAR(100) NOT NULL,
    platform VARCHAR(10) NOT NULL,    -- 'MT4' or 'MT5'
    mtapi_endpoint VARCHAR(255) NOT NULL DEFAULT '',
    default_server VARCHAR(100) NOT NULL DEFAULT '',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(code, platform)
);

-- Seed existing brokers from mt_accounts data.
-- Default mtapi_endpoint to empty (= mtapi.io public gateway).
-- Admin can override via UpdateBroker in admin panel later.
INSERT INTO brokers (code, name, platform, mtapi_endpoint, default_server)
SELECT DISTINCT
    regexp_replace(upper(broker_company), '[^A-Z0-9]', '_', 'g'),
    broker_company,
    mt_type,
    '',  -- default: use mtapi.io public gateway
    broker_server
FROM mt_accounts
WHERE NOT is_disabled
ON CONFLICT (code, platform) DO NOTHING;

-- Link accounts to brokers.
ALTER TABLE mt_accounts ADD COLUMN IF NOT EXISTS broker_id UUID REFERENCES brokers(id);

UPDATE mt_accounts a
SET broker_id = b.id
FROM brokers b
WHERE b.code = regexp_replace(upper(a.broker_company), '[^A-Z0-9]', '_', 'g')
  AND a.broker_id IS NULL;

-- Update view: resolve mtapi gateway endpoint from broker config.
-- Empty mtapi_endpoint → gateway code falls back to mtapi.io public gateway.
DROP VIEW IF EXISTS mt_accounts_v2;
CREATE VIEW mt_accounts_v2 AS
SELECT
    a.id,
    a.user_id,
    a.mt_type AS platform,
    a.broker_company AS broker,
    COALESCE(NULLIF(b.mtapi_endpoint, ''), '')::varchar(100) AS mtapi_host,
    a.mtapi_port,
    a.login,
    a.password,
    a.mt_token,
    a.broker_host,
    a.broker_server AS server,
    NOT a.is_disabled AS is_active,
    a.canonical_subscribed_symbols,
    a.created_at,
    a.updated_at
FROM mt_accounts a
LEFT JOIN brokers b ON a.broker_id = b.id
WHERE NOT a.is_disabled;
