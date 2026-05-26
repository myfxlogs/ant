-- 112_brokers.down.sql
DROP VIEW IF EXISTS mt_accounts_v2;
ALTER TABLE mt_accounts DROP COLUMN IF EXISTS broker_id;
DROP TABLE IF EXISTS brokers;

-- Restore original view (without broker join)
CREATE OR REPLACE VIEW mt_accounts_v2 AS
SELECT
    id,
    user_id,
    mt_type AS platform,
    broker_company AS broker,
    broker_host AS mtapi_host,
    mtapi_port,
    login,
    password,
    mt_token,
    broker_server AS server,
    NOT is_disabled AS is_active,
    canonical_subscribed_symbols,
    created_at,
    updated_at
FROM mt_accounts
WHERE NOT is_disabled;
