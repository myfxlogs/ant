-- 189_drop_plaintext_credentials: drop plaintext password/mt_token columns.
-- Run AFTER application backfill has encrypted all existing plaintext credentials.
-- Prerequisite: verify password_encrypted IS NOT NULL for all active accounts.

-- Safety check: refuse to run if any account still has unencrypted password.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM mt_accounts
        WHERE password_encrypted IS NULL
          AND password IS NOT NULL
          AND password <> ''
          AND deleted_at IS NULL
    ) THEN
        RAISE EXCEPTION 'Cannot drop plaintext columns: some accounts still have unencrypted passwords. Run application backfill first.';
    END IF;
END $$;

ALTER TABLE mt_accounts DROP COLUMN IF EXISTS password;
ALTER TABLE mt_accounts DROP COLUMN IF EXISTS mt_token;
