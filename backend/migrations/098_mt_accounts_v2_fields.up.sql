-- 098_mt_accounts_v2_fields.up.sql
-- Add v2 columns to mt_accounts and create the v2 view.
-- V1 fields are preserved read-only; removed at M9.

ALTER TABLE mt_accounts
    ADD COLUMN IF NOT EXISTS mtapi_port TEXT NOT NULL DEFAULT '443',
    ADD COLUMN IF NOT EXISTS mtapi_token_encrypted BYTEA,
    ADD COLUMN IF NOT EXISTS password_encrypted BYTEA,
    ADD COLUMN IF NOT EXISTS canonical_subscribed_symbols TEXT[]
        NOT NULL DEFAULT ARRAY[]::TEXT[];

CREATE OR REPLACE VIEW mt_accounts_v2 AS
SELECT
    id, user_id,
    mt_type AS platform,
    broker_company AS broker,
    broker_host AS mtapi_host,
    mtapi_port,
    login,
    password_encrypted,
    mtapi_token_encrypted,
    broker_server AS server,
    NOT is_disabled AS is_active,
    canonical_subscribed_symbols,
    created_at, updated_at
FROM mt_accounts
WHERE NOT is_disabled;
