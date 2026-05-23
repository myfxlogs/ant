-- 061_user_ai_primary.down.sql
BEGIN;

ALTER TABLE users
    DROP COLUMN IF EXISTS ai_primary_provider_id,
    DROP COLUMN IF EXISTS ai_primary_model;

COMMIT;
