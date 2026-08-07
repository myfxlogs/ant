-- 098_mt_accounts_v2_fields.down.sql
DROP VIEW IF EXISTS mt_accounts_v2;
ALTER TABLE mt_accounts
  DROP COLUMN IF EXISTS mtapi_port,
  DROP COLUMN IF EXISTS mtapi_token_encrypted,
  DROP COLUMN IF EXISTS password_encrypted,
  DROP COLUMN IF EXISTS canonical_subscribed_symbols;
