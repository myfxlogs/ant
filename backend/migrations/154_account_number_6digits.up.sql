-- Migration 154: Expand account_number from VARCHAR(5) to VARCHAR(6).
-- Existing 5-digit numbers are preserved. New users get 6-digit numbers.
-- Capacity: 7 × 8⁵ = 229,376 (up from 28,672).
BEGIN;

ALTER TABLE users ALTER COLUMN account_number TYPE VARCHAR(6);

COMMIT;
