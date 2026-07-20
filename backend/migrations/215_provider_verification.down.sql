-- 215_provider_verification.down.sql
DROP TABLE IF EXISTS provider_verification_requests CASCADE;
ALTER TABLE users DROP COLUMN IF EXISTS provider_type;
ALTER TABLE users DROP COLUMN IF EXISTS verified_provider;
