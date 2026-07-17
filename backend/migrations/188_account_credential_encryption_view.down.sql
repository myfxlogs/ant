-- Revert mt_accounts_v2 to expose plaintext columns (rollback).
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
    (a.account_status <> 'frozen') AS is_active,
    a.canonical_subscribed_symbols,
    a.created_at,
    a.updated_at
FROM mt_accounts a
LEFT JOIN brokers b ON a.broker_id = b.id;
