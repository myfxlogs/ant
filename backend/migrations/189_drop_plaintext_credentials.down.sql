-- Restore plaintext columns (rollback).
ALTER TABLE mt_accounts ADD COLUMN IF NOT EXISTS password TEXT NOT NULL DEFAULT '';
ALTER TABLE mt_accounts ADD COLUMN IF NOT EXISTS mt_token TEXT;
