-- Migration 154 rollback: shrink back to VARCHAR(5).
-- Fails if any 6-digit numbers exist.
BEGIN;

ALTER TABLE users ALTER COLUMN account_number TYPE VARCHAR(5);

COMMIT;
