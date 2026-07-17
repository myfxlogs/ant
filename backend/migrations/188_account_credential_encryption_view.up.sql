-- 188_account_credential_encryption_view: update mt_accounts_v2 to expose encrypted columns
-- The view now exposes password_encrypted (BYTEA) and mtapi_token_encrypted (BYTEA)
-- instead of plaintext password (TEXT) and mt_token (TEXT).
-- Plaintext columns are retained temporarily for application-level backfill.

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
    a.password_encrypted,
    a.mtapi_token_encrypted,
    a.broker_host,
    a.broker_server AS server,
    (a.account_status <> 'frozen') AS is_active,
    a.canonical_subscribed_symbols,
    a.created_at,
    a.updated_at
FROM mt_accounts a
LEFT JOIN brokers b ON a.broker_id = b.id;
